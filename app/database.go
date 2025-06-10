// database.go
package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var db *sql.DB

func initDB() error {
	var err error
	connStr := "postgres://postgres:postgres@db:5432/batches?sslmode=disable"
	
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("DB connection error: %v, retrying...", err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("DB ping error: %v, retrying...", err)
		time.Sleep(2 * time.Second)
	}
	
	if err != nil {
		return err
	}

	// Создаем таблицы
	if err := createTables(); err != nil {
		return err
	}

	// Создаем директорию для результатов
	if err := os.Mkdir("results", 0755); err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func createTables() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS run_history (
			id SERIAL PRIMARY KEY,
			filename TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			output_path TEXT NOT NULL,
			host TEXT
		)
	`)
	if err != nil {
		return err
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
	return err
}