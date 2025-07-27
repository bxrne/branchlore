#!/bin/bash

echo "🌳 BranchLore SQLite Database Server Example"
echo "============================================"

# Clean up any existing data
rm -rf ./data
mkdir -p ./data

echo
echo "1. Initialize a new database called 'myapp'"
./branchlore init myapp

echo
echo "2. Start the server in the background"
./branchlore server &
SERVER_PID=$!
sleep 2

echo
echo "3. Create a feature branch"
./branchlore branch create myapp feature-users

echo
echo "4. List all branches"
./branchlore branch list myapp

echo
echo "5. Test connection and create a table on main branch"
echo "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT);" | ./branchlore connect myapp@main

echo
echo "6. Insert some data on main branch"
echo "INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com');" | ./branchlore connect myapp@main

echo
echo "7. Query data from main branch"
echo "SELECT * FROM users;" | ./branchlore connect myapp@main

echo
echo "8. Create different data on feature branch"
echo "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT, role TEXT);" | ./branchlore connect myapp@feature-users
echo "INSERT INTO users (name, email, role) VALUES ('Charlie', 'charlie@example.com', 'admin');" | ./branchlore connect myapp@feature-users

echo
echo "9. Query data from feature branch"
echo "SELECT * FROM users;" | ./branchlore connect myapp@feature-users

echo
echo "10. Query data from main branch again (should be different)"
echo "SELECT * FROM users;" | ./branchlore connect myapp@main

echo
echo "11. Clean up"
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo
echo "✅ Example completed! Each branch maintains its own database state."
echo "   This demonstrates Git-like branching for SQLite databases."