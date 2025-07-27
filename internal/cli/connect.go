package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewConnectCmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "connect [database@branch]",
		Short: "Connect to a database branch and execute SQL",
		Long: `Connect to a database branch and execute SQL queries.
Connection format: database@branch (e.g., mydb@feature-1)
If no branch is specified, defaults to 'main'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connStr := args[0]
			parts := strings.Split(connStr, "@")

			dbName := parts[0]
			branch := "main"
			if len(parts) > 1 {
				branch = parts[1]
			}

			fmt.Printf("Connected to %s@%s\n", dbName, branch)
			fmt.Printf("Server: %s\n", serverURL)
			fmt.Println("Type 'exit' or 'quit' to exit")
			fmt.Println("Type SQL queries to execute them")
			fmt.Println()

			scanner := bufio.NewScanner(os.Stdin)
			for {
				fmt.Printf("%s@%s> ", dbName, branch)
				if !scanner.Scan() {
					break
				}

				query := strings.TrimSpace(scanner.Text())
				if query == "" {
					continue
				}

				if query == "exit" || query == "quit" {
					break
				}

				if err := executeQuery(serverURL, dbName, branch, query); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "BranchLore server URL")

	return cmd
}

func executeQuery(serverURL, dbName, branch, query string) error {
	data := url.Values{}
	data.Set("query", query)

	queryURL := fmt.Sprintf("%s/query?db=%s&branch=%s", serverURL, dbName, branch)

	resp, err := http.Post(queryURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if errorMsg, exists := result["error"]; exists {
		return fmt.Errorf("query error: %v", errorMsg)
	}

	if _, exists := result["columns"]; exists {
		printQueryResult(result)
	} else {
		printModifyResult(result)
	}

	return nil
}

func printQueryResult(result map[string]interface{}) {
	columns := result["columns"].([]interface{})
	rows := result["rows"].([]interface{})

	if len(rows) == 0 {
		fmt.Println("No results")
		return
	}

	var buf bytes.Buffer
	for i, col := range columns {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(fmt.Sprintf("%-15s", col))
	}
	fmt.Println(buf.String())

	buf.Reset()
	for i := range columns {
		if i > 0 {
			buf.WriteString("-+-")
		}
		buf.WriteString(strings.Repeat("-", 15))
	}
	fmt.Println(buf.String())

	for _, row := range rows {
		rowData := row.([]interface{})
		buf.Reset()
		for i, val := range rowData {
			if i > 0 {
				buf.WriteString(" | ")
			}
			buf.WriteString(fmt.Sprintf("%-15v", val))
		}
		fmt.Println(buf.String())
	}

	fmt.Printf("\n%d rows returned\n", len(rows))
}

func printModifyResult(result map[string]interface{}) {
	if rowsAffected, exists := result["rows_affected"]; exists {
		fmt.Printf("Rows affected: %v\n", rowsAffected)
	}
	if lastInsertId, exists := result["last_insert_id"]; exists {
		if id := lastInsertId.(float64); id > 0 {
			fmt.Printf("Last insert ID: %.0f\n", id)
		}
	}
}
