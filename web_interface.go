package main

import (
	"bufio"
	"embed"
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
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/run", runHandler)
	http.HandleFunc("/list", listHandler)
	
	// Serve static files from embedded FS
	staticSubFS, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))

	log.Println("Starting web server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
	// Разрешаем запросы с любого источника
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	output, err := RunBatFile(filepath.Join("batfiles", file))
	if err != nil {
		http.Error(w, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(output))
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

func RunBatFile(filePath string) (string, error) {
	var output strings.Builder
	log.Printf("Running bat file: %s", filePath)

	conn, err := net.Dial("tcp", "localhost:4545")
	if err != nil {
		return "", fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	// Увеличим таймаут
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Ping server
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		return "", fmt.Errorf("ping error: %w", err)
	}

	pingResp, err := readFullResponse(conn)
	if err != nil {
		return "", fmt.Errorf("ping read error: %w", err)
	}
	output.WriteString("PING RESPONSE: " + pingResp + "\n")

	// Open bat file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("file open error: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cmd := scanner.Text()
		output.WriteString("SENDING: " + cmd + "\n")

		if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
			return output.String(), fmt.Errorf("command send error: %w", err)
		}

		resp, err := readFullResponse(conn)
		if err != nil {
			return output.String(), fmt.Errorf("response error: %w", err)
		}
		output.WriteString("RESPONSE: " + resp + "\n")
	}

	return output.String(), nil
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