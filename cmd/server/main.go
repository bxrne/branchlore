package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bxrne/branchlore/internal/server"
)

func main() {
	var (
		port     = flag.String("port", "8080", "Port to listen on")
		dataDir  = flag.String("data-dir", "./data", "Directory to store database files")
		logLevel = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	config := &server.Config{
		Port:     *port,
		DataDir:  *dataDir,
		LogLevel: *logLevel,
	}

	srv, err := server.New(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	fmt.Printf("BranchLore server starting on port %s\n", *port)
	fmt.Printf("Data directory: %s\n", *dataDir)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down server...")
	srv.Shutdown()
}
