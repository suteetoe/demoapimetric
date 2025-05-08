cd /Users/toe/DEV/goproject/demoapimetric/k6
./run-auth-tests.sh


# Run with a specific number of virtual users
./run-auth-tests.sh --vus 50

# Run for a specific duration
./run-auth-tests.sh --duration 5m

# Export results as JSON
./run-auth-tests.sh --output-json

# Combine multiple options
./run-auth-tests.sh --vus 30 --duration 3m --output-csv