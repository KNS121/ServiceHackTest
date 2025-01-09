package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strings"
    "time"
)

func main() {
    // Подключаемся к серверу
    conn, err := net.Dial("tcp", "localhost:4545")
    if err != nil {
        fmt.Println("Connection error:", err)
        return
    }
    defer func() {
        if err := conn.Close(); err != nil {
            fmt.Println("Close connection error:", err)
        }
    }()

    // Отправляем команду пинг на сервер
    pingCommand := "ping"
    //fmt.Println(":", pingCommand)
    _, err = conn.Write([]byte(pingCommand + "\n"))
    if err != nil {
        fmt.Println("Error of ping:", err)
        return
    }

    // Ждем ответа на команду пинг
    pingResponse, err := readFullResponse(conn)
    if err != nil {
        fmt.Println("Error read server message:", err)
        return
    }
    fmt.Println("Get server message ping:", pingResponse)

    // Открываем лог-файл для записи
    logFile, err := os.Create("log.txt")
    if err != nil {
        fmt.Println("Error creation log file:", err)
        return
    }
    defer logFile.Close()

    logWriter := bufio.NewWriter(logFile)
    defer logWriter.Flush()

    // Определяем папку с .bat файлами (относительный путь)
    directory := "batfiles"

    // Итерируемся по файлам в папке
    err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Ext(path) == ".bat" {
            // Открываем .bat файл для чтения
            file, err := os.Open(path)
            if err != nil {
                fmt.Println("Error open bat file:", err)
                return nil
            }
            defer file.Close()

            // Читаем команды из файла и отправляем их на сервер
            scanner := bufio.NewScanner(file)
            for scanner.Scan() {
                command := scanner.Text()
                fmt.Println("Sending command:", command)
                logWriter.WriteString(fmt.Sprintf("Sending command: %s\n", command))

                _, err = conn.Write([]byte(command + "\n"))
                if err != nil {
                    fmt.Println("Error server command:", err)
                    return nil
                }

                // Ждем полного ответа от сервера
                response, err := readFullResponse(conn)
                if err != nil {
                    fmt.Println("Error read server message:", err)
                    return nil
                }
                fmt.Println("Server message:", response)
                logWriter.WriteString(fmt.Sprintf("Server message: %s\n", response))
            }

            if err := scanner.Err(); err != nil {
                fmt.Println("Error reading file:", err)
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println("Error catalog:", err)
    }
}

// Функция для чтения полного ответа от сервера с тайм-аутом
func readFullResponse(conn net.Conn) (string, error) {
    var response strings.Builder
    reader := bufio.NewReader(conn)
    timeout := 2 * time.Second // Устанавливаем тайм-аут на 2 секунды
    conn.SetReadDeadline(time.Now().Add(timeout))

    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
                // Тайм-аут истек, считаем, что ответ завершен
                break
            }
            return "", err
        }
        response.WriteString(line)
    }
    return response.String(), nil
}
