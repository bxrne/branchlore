package main

import (
	"fmt"
	"os"

	"github.com/bxrne/branchlore/internal/cli"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "branchlore",
	Short: "A Git-inspired SQLite database server with branching capabilities",
	Long: `BranchLore is a SQLite database server that provides Git-like branching functionality.
You can create branches, switch between them, and merge changes just like with Git.

Connection strings support branch specification: database.db@branch-name`,
}

func init() {
	rootCmd.AddCommand(cli.NewServerCmd())
	rootCmd.AddCommand(cli.NewBranchCmd())
	rootCmd.AddCommand(cli.NewConnectCmd())
	rootCmd.AddCommand(cli.NewInitCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
