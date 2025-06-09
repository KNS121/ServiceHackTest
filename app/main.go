// main.go
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Инициализация базы данных
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Запуск монитора хостов
	startHostMonitor()

	// Настройка маршрутов
	SetupRoutes()

	// Создаем директорию для результатов
	if err := os.Mkdir("results", 0755); err != nil && !os.IsExist(err) {
		log.Fatal("Failed to create results directory:", err)
	}

	log.Println("Starting web server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}