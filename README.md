# Branchlore 🌳

> **Git-like branching for SQLite databases**

Branchlore lets you create branches of your SQLite database, just like Git branches for code. Experiment with different data changes, test new features, or work on multiple versions simultaneously—all without affecting your main database.

## 🚀 What makes Branchlore special?

- **🌿 Branch your data**: Create independent database branches that work exactly like Git branches
- **🔒 True isolation**: Each branch has its own SQLite file—no shared state, no conflicts
- **⚡ Instant switching**: Connect to any branch instantly with `mydb@branch-name`
- **🛠️ Developer-friendly**: CLI tools and HTTP API for easy integration
- **📦 Zero dependencies**: Just Go and SQLite—no complex setup required

## 📋 Prerequisites

- Go 1.24+ (for building from source)
- Git (used internally for branch management)

## ⚡ Quick Start

### 1. Build Branchlore

```bash
# Clone the repository
git clone https://github.com/bxrne/branchlore
cd branchlore

# Build the binary
go build -o branchlore ./cmd/branchlore/
```

### 2. Try the Example

The fastest way to see Branchlore in action:

```bash
./example.sh
```

This runs a complete demo showing database initialization, branching, and isolated data operations.

### 3. Manual Setup

**Initialize your first database:**
```bash
./branchlore init myapp
```

**Start the server:**
```bash
./branchlore server &
```

**Create a feature branch:**
```bash
./branchlore branch create myapp feature-users
```

**Connect and run SQL:**
```bash
./branchlore connect myapp@main
# Now you're in an interactive SQL session!
# Try: CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
```

## 💡 Real-World Example

Imagine you're building an e-commerce app and want to test a new user authentication system:

```bash
# Start with your main database
./branchlore init ecommerce
./branchlore server &

# Create a branch for your new feature
./branchlore branch create ecommerce auth-redesign

# Work on main branch (current production data)
echo "SELECT COUNT(*) FROM users;" | ./branchlore connect ecommerce@main
# Returns: 1000 users

# Experiment on your feature branch
echo "ALTER TABLE users ADD COLUMN oauth_provider TEXT;" | ./branchlore connect ecommerce@auth-redesign
echo "UPDATE users SET oauth_provider = 'google' WHERE email LIKE '%@gmail.com';" | ./branchlore connect ecommerce@auth-redesign

# Check your changes
echo "SELECT COUNT(*) FROM users WHERE oauth_provider IS NOT NULL;" | ./branchlore connect ecommerce@auth-redesign
# Returns: 300 users

# Main branch is completely unaffected!
echo "PRAGMA table_info(users);" | ./branchlore connect ecommerce@main
# oauth_provider column doesn't exist here
```

## 🔧 Command Reference

### Database Management

```bash
# Initialize a new database
./branchlore init <database-name>

# Example
./branchlore init myproject
```

### Server Operations

```bash
# Start server (default port 8080)
./branchlore server

# Custom configuration
./branchlore server --port 9000 --data-dir /path/to/data --log-level debug
```

### Branch Management

```bash
# Create a new branch
./branchlore branch create <database> <branch-name>

# List all branches
./branchlore branch list <database>

# Delete a branch (cannot delete 'main')
./branchlore branch delete <database> <branch-name>

# Examples
./branchlore branch create myproject feature-payments
./branchlore branch list myproject
./branchlore branch delete myproject old-experiment
```

### Database Connections

```bash
# Connect to main branch
./branchlore connect myproject
./branchlore connect myproject@main

# Connect to specific branch
./branchlore connect myproject@feature-payments

# Connect to different server
./branchlore connect myproject@main --server http://localhost:9000
```

**Interactive SQL Session:**
```sql
myproject@main> CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL);
Rows affected: 0

myproject@main> INSERT INTO products (name, price) VALUES ('Widget', 9.99);
Rows affected: 1
Last insert ID: 1

myproject@main> SELECT * FROM products;
id              | name            | price          
----------------+-----------------+----------------
1               | Widget          | 9.99           

1 rows returned

myproject@main> exit
```

## 🌐 HTTP API

Branchlore provides a REST API for programmatic access:

### Execute Queries

```bash
# Run a SELECT query
curl -X POST "http://localhost:8080/query?db=myproject&branch=main" \
  -d "query=SELECT * FROM users LIMIT 5"

# Run an INSERT
curl -X POST "http://localhost:8080/query?db=myproject&branch=feature-users" \
  -d "query=INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com')"
```

### Branch Management via API

```bash
# Create branch
curl -X POST "http://localhost:8080/branch?db=myproject&action=create&branch=new-feature"

# List branches
curl "http://localhost:8080/branch?db=myproject&action=list"

# Delete branch
curl -X POST "http://localhost:8080/branch?db=myproject&action=delete&branch=old-feature"

# Health check
curl "http://localhost:8080/health"
```

## 🏗️ How It Works

Branchlore combines several technologies to create a seamless branching experience:

1. **Git Repository**: Each database is a Git repository
2. **Git Worktrees**: Each branch gets its own working directory
3. **SQLite Files**: Each branch has its own `main.db` file
4. **HTTP Server**: Handles SQL queries and branch operations
5. **CLI Client**: Provides easy command-line access

```
data/
└── myproject/               # Git repository root
    ├── .git/               # Git metadata
    ├── main.db             # Main branch SQLite file
    └── worktrees/
        ├── feature-users/
        │   └── main.db     # Feature branch SQLite file
        └── feature-payments/
            └── main.db     # Another branch SQLite file
```

## 🎯 Use Cases

**🧪 Feature Development**
- Test database schema changes safely
- Experiment with data transformations
- Develop new features with isolated data

**📊 Data Analysis**
- Create analysis branches for different reports
- Preserve original data while exploring
- Share analysis branches with team members

**🐛 Bug Investigation**
- Reproduce bugs with specific data states
- Test fixes without affecting production data
- Maintain multiple test scenarios

**🚀 Deployment Testing**
- Test migrations on production-like data
- Validate data changes before deployment
- Rollback capabilities with branch switching

## ⚠️ Limitations

- **Single Server**: Each database instance runs on one server (no clustering)
- **File-based Storage**: Uses local file system (no cloud storage integration yet)
- **SQLite Limits**: Inherits SQLite's limitations (single writer, file size, etc.)
- **Branch Merging**: No automatic merge capabilities (manual data migration required)


**Development Setup:**
```bash
git clone https://github.com/bxrne/branchlore
cd branchlore
go mod download
go build -o branchlore ./cmd/branchlore/
./example.sh  # Test your changes
```


