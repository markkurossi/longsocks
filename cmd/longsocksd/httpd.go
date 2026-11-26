//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed templates/index.html static/style.css
var templateFS embed.FS

type DialdX struct {
	ID             string
	Address        string
	Version        string
	ConnectedSince time.Time
	LastSeen       time.Time
	Healthy        bool
	Connections    int
	BytesIn        uint64
	BytesOut       uint64
}

type Connection struct {
	ID          string
	Diald       string
	Client      string
	Destination string
	Started     time.Time
	BytesIn     uint64
	BytesOut    uint64
	State       string
}

type StatsPage struct {
	GeneratedAt time.Time
	Dialds      []*Diald
	Connections []Connection
	NumHealthy  int
}

type HTTPD struct {
	server *http.Server
}

func NewHTTPD() (*HTTPD, error) {
	funcMap := template.FuncMap{
		"bytes": formatBytes,
	}
	tmpl := template.Must(template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html"))

	mux := http.NewServeMux()
	staticFS, err := fs.Sub(templateFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		err := tmpl.Execute(w, getStats())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return &HTTPD{
		server: &http.Server{
			Addr:              config.Longsocksd.HTTP.Listen,
			Handler:           logging(mux),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

func getStats() *StatsPage {
	// Replace these demo values with your actual Longsocks registry/metrics.
	now := time.Now()

	fmt.Printf("hosts: %v\n", hosts)

	var dialds []*Diald
	var numHealthy int

	m.Lock()
	for _, diald := range hosts {
		dialds = append(dialds, diald)
		if diald.Healthy {
			numHealthy++
		}
	}
	m.Unlock()

	dialdsX := []DialdX{
		{
			ID:             "builder",
			Address:        "192.0.2.41:48321",
			Version:        "0.1.0",
			ConnectedSince: now.Add(-18 * time.Hour),
			LastSeen:       now.Add(-2 * time.Second),
			Healthy:        true,
			Connections:    2,
			BytesIn:        184 * 1024 * 1024,
			BytesOut:       91 * 1024 * 1024,
		},
		{
			ID:             "nas",
			Address:        "192.0.2.77:39112",
			Version:        "0.1.0",
			ConnectedSince: now.Add(-6 * 24 * time.Hour),
			LastSeen:       now.Add(-4 * time.Second),
			Healthy:        true,
			Connections:    1,
			BytesIn:        4 * 1024 * 1024,
			BytesOut:       12 * 1024 * 1024,
		},
		{
			ID:             "riscv",
			Address:        "192.0.2.91:44701",
			Version:        "0.1.0",
			ConnectedSince: now.Add(-3 * time.Minute),
			LastSeen:       now.Add(-45 * time.Second),
			Healthy:        false,
			Connections:    0,
			BytesIn:        0,
			BytesOut:       0,
		},
	}
	_ = dialdsX

	connections := []Connection{
		{
			ID:          "c-1042",
			Diald:       "builder",
			Client:      "10.10.0.15:53218",
			Destination: "builder.lab.ephemelier.com:22",
			Started:     now.Add(-27 * time.Minute),
			BytesIn:     83 * 1024 * 1024,
			BytesOut:    21 * 1024 * 1024,
			State:       "active",
		},
		{
			ID:          "c-1041",
			Diald:       "builder",
			Client:      "10.10.0.15:53174",
			Destination: "builder.lab.ephemelier.com:22",
			Started:     now.Add(-2 * time.Hour),
			BytesIn:     101 * 1024 * 1024,
			BytesOut:    70 * 1024 * 1024,
			State:       "active",
		},
		{
			ID:          "c-1040",
			Diald:       "nas",
			Client:      "10.10.0.20:49822",
			Destination: "nas.lab.ephemelier.com:22",
			Started:     now.Add(-11 * time.Minute),
			BytesIn:     4 * 1024 * 1024,
			BytesOut:    12 * 1024 * 1024,
			State:       "active",
		},
	}

	return &StatsPage{
		GeneratedAt: time.Now(),
		Dialds:      dialds,
		Connections: connections,
		NumHealthy:  numHealthy,
	}
}

func (httpd *HTTPD) Run() {
	log.Printf("Longsocks statistics: http://localhost%s", httpd.server.Addr)
	log.Fatal(httpd.server.ListenAndServe())
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for n >= div*unit && exp < 5 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
