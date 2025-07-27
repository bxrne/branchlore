# BranchLore

A Git-inspired SQLite database server that allows you to create branches, merge them, and manage your data in a version-controlled manner. All via Git worktree functionality wrapped around Git and SQLite provided as a daemon for server and a CLI to interact with it.

## Features

- **Branching**: Create branches of your database to experiment without affecting the main data
- **Git Worktrees**: Uses Git worktrees to manage separate database files for each branch
- **SQLite**: Uses SQLite as the underlying database engine
- **CLI**: Command-line interface for easy interaction
- **Connection Strings**: Support branch specification in connection strings (e.g., `mydb@feature-1`)
- **HTTP API**: RESTful API for database operations and branch management

## Installation

```bash
go build ./cmd/branchlore
```

## Quick Start

1. **Initialize a new database:**
   ```bash
   ./branchlore init myapp
   ```

2. **Start the server:**
   ```bash
   ./branchlore server --port 8080 --data-dir ./data
   ```

3. **Create a branch:**
   ```bash
   ./branchlore branch create myapp feature-users
   ```

4. **Connect and query:**
   ```bash
   ./branchlore connect myapp@main
   # Or connect to a specific branch
   ./branchlore connect myapp@feature-users
   ```

## Usage

### CLI Commands

#### Initialize Database
```bash
./branchlore init [database-name]
```

#### Start Server
```bash
./branchlore server [flags]
  -p, --port string       Port to listen on (default "8080")
  -d, --data-dir string   Directory to store database files (default "./data")
  -l, --log-level string  Log level (default "info")
```

#### Branch Management
```bash
# Create a branch
./branchlore branch create [database-name] [branch-name]

# Delete a branch
./branchlore branch delete [database-name] [branch-name]

# List branches
./branchlore branch list [database-name]
```

#### Connect to Database
```bash
./branchlore connect [database@branch]
```

### Connection String Format

BranchLore supports Git-like branch specification in connection strings:

- `mydb` - connects to main branch
- `mydb@main` - explicitly connects to main branch  
- `mydb@feature-1` - connects to feature-1 branch

### HTTP API

#### Execute Query
```http
POST /query?db=mydb&branch=main
Content-Type: application/x-www-form-urlencoded

query=SELECT * FROM users;
```

#### Branch Operations
```http
# Create branch
POST /branch?db=mydb&action=create&branch=feature-1

# Delete branch
POST /branch?db=mydb&action=delete&branch=feature-1

# List branches
GET /branch?db=mydb&action=list
```

## Example

Run the example script to see BranchLore in action:

```bash
./example.sh
```

This demonstrates:
- Database initialization
- Branch creation and management
- Separate data storage per branch
- SQL operations on different branches

## Architecture

BranchLore combines:
- **Git repositories** for version control and branching
- **Git worktrees** for separate working directories per branch
- **SQLite** for the actual database storage
- **HTTP server** for client connections
- **CLI** for administration and direct access

Each database branch gets its own SQLite file in a separate Git worktree, allowing true isolation between branches while maintaining the familiar Git workflow.

## License

MIT
