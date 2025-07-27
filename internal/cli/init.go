package cli

import (
	"fmt"
	"path/filepath"

	"github.com/bxrne/branchlore/internal/git"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "init [database-name]",
		Short: "Initialize a new database with Git repository",
		Long:  "Initialize a new database with Git repository for branching capabilities",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName := args[0]

			gitMgr, err := git.NewManager(dataDir)
			if err != nil {
				return fmt.Errorf("failed to create git manager: %w", err)
			}

			if err := gitMgr.InitDatabase(dbName); err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}

			dbPath := filepath.Join(dataDir, dbName)
			fmt.Printf("Initialized database '%s' at %s\n", dbName, dbPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "./data", "Directory to store database files")

	return cmd
}
