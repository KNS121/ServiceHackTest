// handlers.go
package main

import (

	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

    timestamp := time.Now().Format("20060102_150405")
    safeHost := strings.ReplaceAll(host, ".", "_")
    safeHost = strings.ReplaceAll(safeHost, ":", "_")
    resultFilename := fmt.Sprintf("%s_%s_%s.log", timestamp, safeHost, strings.TrimSuffix(file, ".bat"))
    resultPath := filepath.Join("results", resultFilename)
    
    if err := os.WriteFile(resultPath, []byte(output), 0644); err != nil {
        log.Printf("Failed to save result: %v", err)
    }

    _, err = db.Exec(
        "INSERT INTO run_history (filename, success, output_path, host) VALUES ($1, $2, $3, $4)",
        file, success, resultFilename, host,
    )
    if err != nil {
        log.Printf("Failed to save to DB: %v", err)
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": success,
        "output":  output,
        "log_file": resultFilename,
        "host":    host,
    })
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

func historyHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT 
            rh.id, 
            rh.filename, 
            rh.success, 
            rh.timestamp, 
            rh.output_path,
            rh.host
        FROM run_history rh
        ORDER BY rh.timestamp DESC
    `)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []RunHistory
	for rows.Next() {
		var h RunHistory
		var timestamp time.Time
		err := rows.Scan(&h.ID, &h.Filename, &h.Success, &timestamp, &h.Output, &h.Host)
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

func SetupRoutes() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/run", runHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/history", historyHandler)
	http.HandleFunc("/result", resultHandler)

	http.HandleFunc("/hosts", hostsHandler)
	http.HandleFunc("/hosts/list", listHostsHandler)
	http.HandleFunc("/hosts/add", addHostHandler)
	http.HandleFunc("/hosts/delete", deleteHostHandler)
	
	staticSubFS, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))
}