package main

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    // Подключаемся к серверу
    conn, err := net.Dial("tcp", "localhost:4545")
    if err != nil {
        fmt.Println("Ошибка подключения:", err)
        return
    }
    defer func() {
        if err := conn.Close(); err != nil {
            fmt.Println("Ошибка закрытия соединения:", err)
        }
    }()

    // Определяем папку с .bat файлами (относительный путь)
    directory := "batfiles"

    // Открываем лог-файл для записи
    logFile, err := os.Create("log.txt")
    if err != nil {
        fmt.Println("Ошибка создания лог-файла:", err)
        return
    }
    defer logFile.Close()

    logWriter := bufio.NewWriter(logFile)
    defer logWriter.Flush()

    // Итерируемся по файлам в папке
    err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Ext(path) == ".bat" {
            // Открываем .bat файл для чтения
            file, err := os.Open(path)
            if err != nil {
                fmt.Println("Ошибка открытия файла:", err)
                return nil
            }
            defer file.Close()

            // Читаем команды из файла и отправляем их на сервер
            scanner := bufio.NewScanner(file)
            for scanner.Scan() {
                command := scanner.Text()
                fmt.Println("Отправка команды:", command)
                logWriter.WriteString(fmt.Sprintf("Отправка команды: %s\n", command))

                _, err = conn.Write([]byte(command + "\n"))
                if err != nil {
                    fmt.Println("Ошибка записи на сервер:", err)
                    return nil
                }

                // Ждем полного ответа от сервера
                response, err := readFullResponse(conn)
                if err != nil {
                    fmt.Println("Ошибка чтения с сервера:", err)
                    return nil
                }
                fmt.Println("Получено с сервера:", response)
                logWriter.WriteString(fmt.Sprintf("Получено с сервера: %s\n", response))
            }

            if err := scanner.Err(); err != nil {
                fmt.Println("Ошибка чтения файла:", err)
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println("Ошибка при обходе папки:", err)
    }
}

// Функция для чтения полного ответа от сервера
func readFullResponse(conn net.Conn) (string, error) {
    var response strings.Builder
    reader := bufio.NewReader(conn)
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            return "", err
        }
        response.WriteString(line)
        // Проверяем, что ответ завершен (например, по наличию определенного маркера)
        if strings.HasSuffix(line, "\n") {
            break
        }
    }
    return response.String(), nil
}
