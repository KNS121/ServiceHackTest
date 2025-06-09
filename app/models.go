// models.go
package main

import (
	"embed"
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
	History []RunHistory
}

type RunHistory struct {
	ID        int
	Filename  string
	Success   bool
	Timestamp time.Time
	Output    string
	Host      string
}

type RunResult struct {
	Filename  string
	Success   bool
	Output    string
	Timestamp time.Time
	Host      string
}

type Host struct {
    ID         int        `json:"id"`
    IPAddress  string     `json:"ip_address"`
    Name       string     `json:"name"`
    Status     string     `json:"status"`
    LastChecked *time.Time `json:"last_checked"`
}