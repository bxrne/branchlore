package cli

import (
	"fmt"

	"github.com/bxrne/branchlore/internal/git"
	"github.com/spf13/cobra"
)

func NewBranchCmd() *cobra.Command {
	var dataDir string

	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage database branches",
		Long:  "Create, delete, and list database branches",
	}

	createCmd := &cobra.Command{
		Use:   "create [database-name] [branch-name]",
		Short: "Create a new branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, branchName := args[0], args[1]

			gitMgr, err := git.NewManager(dataDir)
			if err != nil {
				return fmt.Errorf("failed to create git manager: %w", err)
			}

			if err := gitMgr.CreateBranch(dbName, branchName); err != nil {
				return fmt.Errorf("failed to create branch: %w", err)
			}

			fmt.Printf("Created branch '%s' for database '%s'\n", branchName, dbName)
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [database-name] [branch-name]",
		Short: "Delete a branch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName, branchName := args[0], args[1]

			gitMgr, err := git.NewManager(dataDir)
			if err != nil {
				return fmt.Errorf("failed to create git manager: %w", err)
			}

			if err := gitMgr.DeleteBranch(dbName, branchName); err != nil {
				return fmt.Errorf("failed to delete branch: %w", err)
			}

			fmt.Printf("Deleted branch '%s' from database '%s'\n", branchName, dbName)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list [database-name]",
		Short: "List all branches",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbName := args[0]

			gitMgr, err := git.NewManager(dataDir)
			if err != nil {
				return fmt.Errorf("failed to create git manager: %w", err)
			}

			branches, err := gitMgr.ListBranches(dbName)
			if err != nil {
				return fmt.Errorf("failed to list branches: %w", err)
			}

			fmt.Printf("Branches for database '%s':\n", dbName)
			for _, branch := range branches {
				fmt.Printf("  %s\n", branch)
			}
			return nil
		},
	}

	cmd.AddCommand(createCmd, deleteCmd, listCmd)
	cmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "./data", "Directory to store database files")

	return cmd
}
