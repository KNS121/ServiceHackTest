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

func RunBatFile(filePath string) (string, error) {
	var output strings.Builder

	conn, err := net.Dial("tcp", "localhost:4545")
	if err != nil {
		return "", fmt.Errorf("connection error: %w", err)
	}
	defer conn.Close()

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