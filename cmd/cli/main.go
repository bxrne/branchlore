package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/bxrne/branchlore/internal/config"
	"github.com/bxrne/branchlore/internal/database"
	"github.com/bxrne/branchlore/internal/git"
	"github.com/bxrne/branchlore/internal/server"
	"github.com/bxrne/branchlore/internal/simulator"
	"github.com/bxrne/branchlore/internal/storage"
	"github.com/bxrne/branchlore/internal/types"
	"github.com/spf13/cobra"
)

var (
	configPath string
	verbose    bool
	simulate   bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "branchlore",
		Short: "Git-inspired SQLite database server with branching capabilities",
		Long: `Branchlore is a database server that uses Git worktrees to manage
SQLite database branches. Create, merge, and query database branches
like you would with Git.`,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&simulate, "simulate", false, "run in simulation mode")

	rootCmd.AddCommand(
		initCmd(),
		branchCmd(),
		queryCmd(),
		mergeCmd(),
		statusCmd(),
		serveCmd(),
		simulateCmd(),
		schemaCmd(),
		exportCmd(),
		importCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new branchlore repository",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			if len(args) > 0 {
				cfg.RepoPath = args[0]
			}

			fs := storage.NewFileSystem(cfg)
			repo := git.NewRepository(fs.GetRepoPath())

			if err := repo.Init(fs.GetRepoPath()); err != nil {
				log.Fatalf("Failed to initialize repository: %v", err)
			}

			fmt.Printf("Initialized branchlore repository in %s\n", fs.GetRepoPath())
		},
	}
}

func branchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage branches",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all branches",
			Run: func(cmd *cobra.Command, args []string) {
				cfg := loadConfig()
				fs := storage.NewFileSystem(cfg)
				repo := git.NewRepository(fs.GetRepoPath())

				if err := repo.Init(fs.GetRepoPath()); err != nil {
					log.Fatalf("Failed to open repository: %v", err)
				}

				branches, err := repo.ListBranches()
				if err != nil {
					log.Fatalf("Failed to list branches: %v", err)
				}

				for _, branch := range branches {
					marker := " "
					if branch.IsMain {
						marker = "*"
					}
					fmt.Printf("%s %s (%s)\n", marker, branch.Name, branch.Hash[:8])
				}
			},
		},
		&cobra.Command{
			Use:   "create <name>",
			Short: "Create a new branch",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				cfg := loadConfig()
				fs := storage.NewFileSystem(cfg)
				repo := git.NewRepository(fs.GetRepoPath())

				if err := repo.Init(fs.GetRepoPath()); err != nil {
					log.Fatalf("Failed to open repository: %v", err)
				}

				branch, err := repo.CreateBranch(args[0])
				if err != nil {
					log.Fatalf("Failed to create branch: %v", err)
				}

				fmt.Printf("Created branch '%s' (%s)\n", branch.Name, branch.Hash[:8])
			},
		},
		&cobra.Command{
			Use:   "show <name>",
			Short: "Show branch information",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				cfg := loadConfig()
				fs := storage.NewFileSystem(cfg)
				repo := git.NewRepository(fs.GetRepoPath())

				if err := repo.Init(fs.GetRepoPath()); err != nil {
					log.Fatalf("Failed to open repository: %v", err)
				}

				branch, err := repo.GetBranch(args[0])
				if err != nil {
					log.Fatalf("Failed to get branch: %v", err)
				}

				worktreePath := fs.GetWorktreePath(fs.GetRepoPath(), branch.Name)
				dbPath := fs.GetDBPath(worktreePath)

				fmt.Printf("Branch: %s\n", branch.Name)
				fmt.Printf("Hash: %s\n", branch.Hash)
				fmt.Printf("Created: %s\n", branch.CreatedAt.Format(time.RFC3339))
				fmt.Printf("Is Main: %t\n", branch.IsMain)
				fmt.Printf("DB Path: %s\n", dbPath)
				fmt.Printf("DB Exists: %t\n", fs.PathExists(dbPath))

				if fs.PathExists(dbPath) {
					size, err := fs.GetSize(dbPath)
					if err == nil {
						fmt.Printf("DB Size: %d bytes\n", size)
					}
				}
			},
		},
	)

	return cmd
}

func queryCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "query <branch> <sql>",
		Short: "Execute SQL query on a branch's database",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			branch, sqlQuery := args[0], args[1]

			if simulate {
				runQuerySimulation(branch, sqlQuery)
				return
			}

			fs := storage.NewFileSystem(cfg)
			repo := git.NewRepository(fs.GetRepoPath())

			if err := repo.Init(fs.GetRepoPath()); err != nil {
				log.Fatalf("Failed to open repository: %v", err)
			}

			worktreePath, err := repo.CreateWorktree(branch)
			if err != nil {
				log.Fatalf("Failed to create worktree: %v", err)
			}

			dbPath := fs.GetDBPath(worktreePath)
			db := database.NewSQLiteDB()

			if err := db.Open(dbPath); err != nil {
				log.Fatalf("Failed to open database: %v", err)
			}
			defer db.Close()

			if !fs.PathExists(dbPath) {
				if err := db.InitSchema(); err != nil {
					log.Fatalf("Failed to initialize schema: %v", err)
				}
			}

			result, err := db.Query(context.Background(), sqlQuery)
			if err != nil {
				log.Fatalf("Query failed: %v", err)
			}

			printQueryResult(result, output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "output format (table, json)")
	return cmd
}

func mergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge <source> <target>",
		Short: "Merge source branch into target branch",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			source, target := args[0], args[1]

			if simulate {
				fmt.Printf("Simulating merge of '%s' into '%s'\n", source, target)
				fmt.Println("Merge completed successfully (simulated)")
				return
			}

			fs := storage.NewFileSystem(cfg)
			repo := git.NewRepository(fs.GetRepoPath())

			if err := repo.Init(fs.GetRepoPath()); err != nil {
				log.Fatalf("Failed to open repository: %v", err)
			}

			result, err := repo.MergeBranches(source, target)
			if err != nil {
				log.Fatalf("Merge failed: %v", err)
			}

			if result.Success {
				fmt.Printf("Successfully merged '%s' into '%s'\n", source, target)
			} else {
				fmt.Printf("Merge failed: %s\n", result.Message)
				if len(result.Conflicts) > 0 {
					fmt.Println("Conflicts:")
					for _, conflict := range result.Conflicts {
						fmt.Printf("  - %s\n", conflict)
					}
				}
			}
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show repository status",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			fs := storage.NewFileSystem(cfg)
			repo := git.NewRepository(fs.GetRepoPath())

			if err := repo.Init(fs.GetRepoPath()); err != nil {
				log.Fatalf("Failed to open repository: %v", err)
			}

			branches, err := repo.ListBranches()
			if err != nil {
				log.Fatalf("Failed to list branches: %v", err)
			}

			dbPaths, err := fs.GetBranchDBs()
			if err != nil {
				log.Fatalf("Failed to get branch databases: %v", err)
			}

			fmt.Printf("Repository: %s\n", fs.GetRepoPath())
			fmt.Printf("Branches: %d\n", len(branches))
			fmt.Printf("Databases: %d\n", len(dbPaths))
			fmt.Println()

			for _, branch := range branches {
				marker := " "
				if branch.IsMain {
					marker = "*"
				}

				dbStatus := "no database"
				if dbPath, exists := dbPaths[branch.Name]; exists {
					size, err := fs.GetSize(dbPath)
					if err == nil {
						dbStatus = fmt.Sprintf("%d bytes", size)
					}
				}

				fmt.Printf("%s %-20s %s (%s)\n", marker, branch.Name, branch.Hash[:8], dbStatus)
			}
		},
	}
}

func serveCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP API server",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			if addr != "" {
				cfg.ServerAddr = addr
			}

			fs := storage.NewFileSystem(cfg)
			repo := git.NewRepository(fs.GetRepoPath())

			if err := repo.Init(fs.GetRepoPath()); err != nil {
				log.Fatalf("Failed to initialize repository: %v", err)
			}

			srv := server.NewHTTPServer(cfg, repo, fs)
			ctx := context.Background()

			fmt.Printf("Starting server on %s\n", cfg.ServerAddr)
			if err := srv.Start(ctx, cfg.ServerAddr); err != nil {
				log.Fatalf("Server failed: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", "", "server address (default from config)")
	return cmd
}

func simulateCmd() *cobra.Command {
	var scenario string
	cmd := &cobra.Command{
		Use:   "simulate [scenario]",
		Short: "Run simulation tests",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := loadConfig()
			cfg.Simulate = true

			if len(args) > 0 {
				scenario = args[0]
			}
			if scenario == "" {
				scenario = "basic"
			}

			sim := simulator.NewMockSimulator(cfg)
			ctx := context.Background()

			fmt.Printf("Running simulation: %s\n", scenario)
			if err := sim.Run(ctx, scenario); err != nil {
				log.Fatalf("Simulation failed: %v", err)
			}

			metrics := sim.GetMetrics()
			fmt.Println("\nSimulation Results:")
			for key, value := range metrics {
				fmt.Printf("  %s: %v\n", key, value)
			}
		},
	}

	cmd.Flags().StringVarP(&scenario, "scenario", "s", "basic", "simulation scenario (basic, stress, concurrent, chaos)")
	return cmd
}

func schemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage database schema",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "show <branch>",
			Short: "Show database schema for branch",
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				cfg := loadConfig()
				branch := args[0]

				fs := storage.NewFileSystem(cfg)
				repo := git.NewRepository(fs.GetRepoPath())

				if err := repo.Init(fs.GetRepoPath()); err != nil {
					log.Fatalf("Failed to open repository: %v", err)
				}

				worktreePath, err := repo.CreateWorktree(branch)
				if err != nil {
					log.Fatalf("Failed to create worktree: %v", err)
				}

				dbPath := fs.GetDBPath(worktreePath)
				db := database.NewSQLiteDB()

				if err := db.Open(dbPath); err != nil {
					log.Fatalf("Failed to open database: %v", err)
				}
				defer db.Close()

				tables, err := db.GetTables()
				if err != nil {
					log.Fatalf("Failed to get tables: %v", err)
				}

				fmt.Printf("Schema for branch '%s':\n", branch)
				for _, table := range tables {
					schema, err := db.GetSchema(table)
					if err != nil {
						fmt.Printf("  %s: (error getting schema)\n", table)
						continue
					}
					fmt.Printf("  %s: %s\n", table, schema)
				}
			},
		},
	)

	return cmd
}

func exportCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "export <branch> <file>",
		Short: "Export branch data",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Export functionality not yet implemented\n")
			fmt.Printf("Would export branch '%s' to '%s' in format '%s'\n", args[0], args[1], format)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "sql", "export format (sql, json, csv)")
	return cmd
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <branch> <file>",
		Short: "Import data into branch",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Import functionality not yet implemented\n")
			fmt.Printf("Would import from '%s' into branch '%s'\n", args[1], args[0])
		},
	}
}

func loadConfig() *types.Config {
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	config.LoadFromEnv(cfg)

	if simulate {
		cfg.Simulate = true
	}

	if verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	return cfg
}

func printQueryResult(result *types.QueryResult, format string) {
	switch format {
	case "json":
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		}
	case "table":
		if len(result.Columns) == 0 {
			fmt.Println("No results")
			return
		}

		for i, col := range result.Columns {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Printf("%-15s", col)
		}
		fmt.Println()

		for i := 0; i < len(result.Columns); i++ {
			if i > 0 {
				fmt.Print("-+-")
			}
			fmt.Print(strings.Repeat("-", 15))
		}
		fmt.Println()

		for _, row := range result.Rows {
			for i, val := range row {
				if i > 0 {
					fmt.Print(" | ")
				}
				fmt.Printf("%-15v", val)
			}
			fmt.Println()
		}

		fmt.Printf("\n%d rows\n", result.Count)
	}
}

func runQuerySimulation(branch, sql string) {
	fmt.Printf("Simulating query on branch '%s': %s\n", branch, sql)

	time.Sleep(time.Duration(10+len(sql)) * time.Millisecond)

	if strings.Contains(strings.ToUpper(sql), "SELECT") {
		fmt.Println("Results (simulated):")
		fmt.Println("id | msg")
		fmt.Println("---+----")
		fmt.Println("1  | test")
		fmt.Println("2  | hello")
		fmt.Println("\n2 rows")
	} else {
		fmt.Println("Query executed successfully (simulated)")
	}
}
