package main

import (
	"bufio"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type BatFile struct {
	Name string
	Path string
}

type PageData struct {
	Title   string
	BatFiles []BatFile
	History []RunHistory
}

type RunHistory struct {
	ID        int
	Filename  string
	Success   bool
	Timestamp time.Time
	Output    string
}

type RunResult struct {
	Filename  string
	Success   bool
	Output    string
	Timestamp time.Time
}

type Host struct {
    ID         int        `json:"id"`
    IPAddress  string     `json:"ip_address"`
    Name       string     `json:"name"`
    Status     string     `json:"status"`
    LastChecked *time.Time `json:"last_checked"`
}

var db *sql.DB

func main() {
	initDB()
	defer db.Close()

    startHostMonitor()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/run", runHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/history", historyHandler)
	http.HandleFunc("/result", resultHandler)

	http.HandleFunc("/hosts", hostsHandler)
	http.HandleFunc("/hosts/list", listHostsHandler)
	http.HandleFunc("/hosts/add", addHostHandler)
	http.HandleFunc("/hosts/delete", deleteHostHandler)
	
	// Serve static files from embedded FS
	staticSubFS, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))

	// Create results directory
	os.Mkdir("results", 0755)

	log.Println("Starting web server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	var err error
	connStr := "postgres://postgres:postgres@localhost:5432/batches?sslmode=disable"
	// Try to connect with retries
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("DB connection error: %v, retrying...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		err = db.Ping()
		if err == nil {
			break
		}
		log.Printf("DB ping error: %v, retrying...", err)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS run_history (
			id SERIAL PRIMARY KEY,
			filename TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			output_path TEXT NOT NULL
		)
	`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

    _, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS hosts (
        id SERIAL PRIMARY KEY,
        ip_address TEXT NOT NULL,
        name TEXT,
        status TEXT DEFAULT 'unknown',
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        last_checked TIMESTAMP
    )
    `)
	if err != nil {
    	log.Fatal("Failed to create hosts table:", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	batFiles, err := getBatFiles("batfiles")
	if err != nil {
		http.Error(w, "Error reading bat files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:   "Batch Commands Manager",
		BatFiles: batFiles,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Execution error: "+err.Error(), http.StatusInternalServerError)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	batFiles, err := getBatFiles("batfiles")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var names []string
	for _, f := range batFiles {
		names = append(names, f.Name)
	}

	fmt.Fprint(w, strings.Join(names, "|"))
}

func runHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Content-Type", "application/json")

    file := r.URL.Query().Get("file")
    host := r.URL.Query().Get("host")

    if file == "" {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]string{"error": "Missing file parameter"})
        return
    }

    output, success, err := RunBatFile(filepath.Join("batfiles", file), host)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Error: " + err.Error(),
            "success": false,
        })
        return
    }

    // Save result to file
    timestamp := time.Now().Format("20060102_150405")
    resultFilename := fmt.Sprintf("%s_%s.log", timestamp, strings.TrimSuffix(file, ".bat"))
    resultPath := filepath.Join("results", resultFilename)
    
    if err := os.WriteFile(resultPath, []byte(output), 0644); err != nil {
        log.Printf("Failed to save result: %v", err)
    }

    // Save to database
    _, err = db.Exec(
        "INSERT INTO run_history (filename, success, output_path) VALUES ($1, $2, $3)",
        file, success, resultFilename,
    )
    if err != nil {
        log.Printf("Failed to save to DB: %v", err)
    }

    // Return result with success status
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": success,
        "output":  output,
        "log_file": resultFilename,
    })
}

func getBatFiles(dir string) ([]BatFile, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.bat"))
	if err != nil {
		return nil, err
	}

	var batFiles []BatFile
	for _, f := range files {
		batFiles = append(batFiles, BatFile{
			Name: filepath.Base(f),
			Path: f,
		})
	}
	return batFiles, nil
}

func RunBatFile(filePath, host string) (string, bool, error) {
    // Проверяем доступность хоста перед выполнением
    if !pingHost(host) {
        return "", false, fmt.Errorf("host %s is unreachable", host)
    }

    conn, err := net.Dial("tcp", fmt.Sprintf("%s:4545", host))
    if err != nil {
        return "", false, fmt.Errorf("connection error: %w", err)
    }
    defer conn.Close()

    conn.SetDeadline(time.Now().Add(10 * time.Second))

    // Ping server
    if _, err := conn.Write([]byte("ping\n")); err != nil {
        return "", false, fmt.Errorf("ping error: %w", err)
    }

    pingResp, err := readFullResponse(conn)
    if err != nil {
        return "", false, fmt.Errorf("ping read error: %w", err)
    }
    
    var output strings.Builder
    output.WriteString("PING RESPONSE: " + pingResp + "\n")
    
    success := true

    // Open bat file
    file, err := os.Open(filePath)
    if err != nil {
        return "", false, fmt.Errorf("file open error: %w", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        cmd := scanner.Text()
        output.WriteString("SENDING: " + cmd + "\n")

        if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
            return output.String(), false, fmt.Errorf("command send error: %w", err)
        }

        resp, err := readFullResponse(conn)
        if err != nil {
            return output.String(), false, fmt.Errorf("response error: %w", err)
        }
        
        output.WriteString("RESPONSE: " + resp + "\n")
        
        // Check for error in response
        if strings.Contains(resp, "Error executing command") {
            success = false
        }
    }

    return output.String(), success, nil
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, filename, success, timestamp, output_path FROM run_history ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []RunHistory
	for rows.Next() {
		var h RunHistory
		var timestamp time.Time
		err := rows.Scan(&h.ID, &h.Filename, &h.Success, &timestamp, &h.Output)
		if err != nil {
			log.Printf("Error scanning history row: %v", err)
			continue
		}
		h.Timestamp = timestamp
		history = append(history, h)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, filepath.Join("results", file))
}

func readFullResponse(conn net.Conn) (string, error) {
	var sb strings.Builder
	reader := bufio.NewReader(conn)
	
	// Читаем до таймаута или конца потока данных
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			return sb.String(), err
		}
		sb.WriteString(line)
		
		// Если сервер возвращает маркер конца ответа
		if strings.Contains(line, "END_OF_RESPONSE") {
			break
		}
	}
	
	return sb.String(), nil
}

func hostsHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFS(templatesFS, "templates/hosts.html")
    if err != nil {
        http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    data := PageData{
        Title: "Hosts Management",
    }

    if err := tmpl.Execute(w, data); err != nil {
        http.Error(w, "Execution error: "+err.Error(), http.StatusInternalServerError)
    }
}

func listHostsHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, ip_address, name, status, last_checked FROM hosts ORDER BY created_at DESC")
    if err != nil {
        http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var hosts []Host

    for rows.Next() {
        var h Host
        if err := rows.Scan(&h.ID, &h.IPAddress, &h.Name, &h.Status, &h.LastChecked); err != nil {
            log.Printf("Error scanning host row: %v", err)
            continue
        }
        hosts = append(hosts, h)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(hosts)
}

func addHostHandler(w http.ResponseWriter, r *http.Request) {
    var host struct {
        IPAddress string `json:"ip_address"`
        Name      string `json:"name"`
    }

    if err := json.NewDecoder(r.Body).Decode(&host); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    _, err := db.Exec(
        "INSERT INTO hosts (ip_address, name, status) VALUES ($1, $2, $3)",
        host.IPAddress, host.Name, "inactive",
    )
    if err != nil {
        http.Error(w, "Failed to add host: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func deleteHostHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "Missing id parameter", http.StatusBadRequest)
        return
    }

    _, err := db.Exec("DELETE FROM hosts WHERE id = $1", id)
    if err != nil {
        http.Error(w, "Failed to delete host: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func pingHost(host string) bool {
    // Особый случай для localhost
    if host == "localhost" {
        host = "127.0.0.1"
    }
    
    timeout := 2 * time.Second
    address := fmt.Sprintf("%s:4545", host)
    
    // Увеличим логирование для диагностики
    log.Printf("Pinging host: %s", address)
    
    conn, err := net.DialTimeout("tcp", address, timeout)
    if err != nil {
        log.Printf("Ping failed for %s: %v", address, err)
        return false
    }
    defer conn.Close()
    
    // Установим общий таймаут
    conn.SetDeadline(time.Now().Add(timeout))
    
    // Отправляем ping
    if _, err := conn.Write([]byte("ping\n")); err != nil {
        log.Printf("Ping send error to %s: %v", address, err)
        return false
    }
    
    // Читаем ответ
    response := make([]byte, 1024)
    n, err := conn.Read(response)
    if err != nil {
        log.Printf("Ping read error from %s: %v", address, err)
        return false
    }
    
    respStr := strings.TrimSpace(string(response[:n]))
    log.Printf("Ping response from %s: %s", address, respStr)
    
    // Проверяем ответ
    return strings.Contains(respStr, "PONG")
}

func startHostMonitor() {
    ticker := time.NewTicker(3 * time.Second)
    go func() {
        for range ticker.C {
            rows, err := db.Query("SELECT id, ip_address FROM hosts")
            if err != nil {
                log.Printf("Host monitor query error: %v", err)
                continue
            }
            
            var hosts []struct {
                ID  int
                IP  string
            }
            
            for rows.Next() {
                var id int
                var ip string
                if err := rows.Scan(&id, &ip); err != nil {
                    log.Printf("Host scan error: %v", err)
                    continue
                }
                hosts = append(hosts, struct{ID int; IP string}{id, ip})
            }
            rows.Close()
            
            for _, host := range hosts {
                status := "inactive"
                if pingHost(host.IP) {
                    status = "active"
                }
                
                _, err := db.Exec(
                    "UPDATE hosts SET status = $1, last_checked = CURRENT_TIMESTAMP WHERE id = $2",
                    status, host.ID,
                )
                if err != nil {
                    log.Printf("Host update error for %s: %v", host.IP, err)
                } else {
                    log.Printf("Updated host %s status to %s", host.IP, status)
                }
            }
        }
    }()
}