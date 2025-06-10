// hosts.go
package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func pingHost(host string) bool {
    if host == "localhost" {
        host = "host.docker.internal"
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
                hosts = append(hosts, struct{ID int; IP string}{id, ip})
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