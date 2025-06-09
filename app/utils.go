package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	if !pingHost(host) {
		return "", false, fmt.Errorf("host %s is unreachable", host)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:4545", host))
	if err != nil {
		return "", false, fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))

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

		if strings.Contains(resp, "Error executing command") {
			success = false
		}
	}

	return output.String(), success, nil
}

func readFullResponse(conn net.Conn) (string, error) {
	var sb strings.Builder
	reader := bufio.NewReader(conn)

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

		if strings.Contains(line, "END_OF_RESPONSE") {
			break
		}
	}

	return sb.String(), nil
}

func pingHost(host string) bool {
	if host == "localhost" {
		host = "127.0.0.1"
	}

	timeout := 2 * time.Second
	address := fmt.Sprintf("%s:4545", host)

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		log.Printf("Ping failed for %s: %v", address, err)
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write([]byte("ping\n")); err != nil {
		log.Printf("Ping send error to %s: %v", address, err)
		return false
	}

	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		log.Printf("Ping read error from %s: %v", address, err)
		return false
	}

	respStr := strings.TrimSpace(string(response[:n]))
	log.Printf("Ping response from %s: %s", address, respStr)

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
				hosts = append(hosts, struct{ ID int; IP string }{id, ip})
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
