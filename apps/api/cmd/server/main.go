package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"

	"co-tuong-wiki-api/internal/httpapi"
	"co-tuong-wiki-api/internal/lessons"
)

func main() {
	repository, err := lessons.LoadRepository(lessons.DefaultDataPath())
	if err != nil {
		log.Fatalf("load lessons: %v", err)
	}

	addr := ":" + env("PORT", "8090")
	server := httpapi.NewServer(repository)

	// Bind the port before logging readiness so startup failures are not reported as a running API.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			log.Fatalf("api port %s is already in use; set PORT to another value, for example PORT=8091", env("PORT", "8090"))
		}
		log.Fatalf("listen on %s: %v", addr, err)
	}
	defer listener.Close()

	log.Printf("co-tuong-wiki api listening on %s", addr)
	if err := http.Serve(listener, server); err != nil {
		log.Fatal(err)
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
