// batfiles.go
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