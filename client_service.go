package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
)

func main() {
    // Открываем .bat файл для чтения
    file, err := os.Open("wmic_script.bat")
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer file.Close()

    // Подключаемся к серверу
    conn, err := net.Dial("tcp", "localhost:4545")
    if err != nil {
        fmt.Println("Error connecting:", err)
        return
    }
    defer func() {
        if err := conn.Close(); err != nil {
            fmt.Println("Error closing connection:", err)
        }
    }()

    // Читаем команды из файла и отправляем их на сервер
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        command := scanner.Text()
        fmt.Println("Sending command:", command)

        _, err = conn.Write([]byte(command + "\n"))
        if err != nil {
            fmt.Println("Error writing to server:", err)
            return
        }

        buf := make([]byte, 1024)
        n, err := conn.Read(buf)
        if err != nil {
            fmt.Println("Error reading from server:", err)
            return
        }
        fmt.Println("Received from server:", string(buf[:n]))
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading file:", err)
    }
}