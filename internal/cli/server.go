package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bxrne/branchlore/internal/server"
	"github.com/spf13/cobra"
)

func NewServerCmd() *cobra.Command {
	var port, dataDir, logLevel string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the BranchLore database server",
		Long:  "Start the BranchLore database server with Git-like branching capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &server.Config{
				Port:     port,
				DataDir:  dataDir,
				LogLevel: logLevel,
			}

			srv, err := server.New(config)
			if err != nil {
				return fmt.Errorf("failed to create server: %w", err)
			}

			go func() {
				if err := srv.Start(); err != nil {
					log.Fatalf("Server failed to start: %v", err)
				}
			}()

			fmt.Printf("BranchLore server starting on port %s\n", port)
			fmt.Printf("Data directory: %s\n", dataDir)

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c

			fmt.Println("\nShutting down server...")
			srv.Shutdown()
			return nil
		},
	}

	cmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "./data", "Directory to store database files")
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	return cmd
}
