#!/bin/bash

# Run k6 load tests for authentication service
# Usage: ./run-auth-tests.sh [options]
#
# Options:
#   --output-json    Output results as JSON
#   --output-csv     Output results as CSV
#   --vus N          Number of virtual users (default: use stages in script)
#   --duration D     Override test duration (e.g. 1m30s)

# Default options
OUTPUT_OPTIONS=""
VUS_OPTION=""
DURATION_OPTION=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --output-json)
      OUTPUT_OPTIONS="--out json=results/result.json"
      mkdir -p results
      shift
      ;;
    --output-csv)
      OUTPUT_OPTIONS="--out csv=results/result.csv"
      mkdir -p results
      shift
      ;;
    --vus)
      VUS_OPTION="--vus $2"
      shift
      shift
      ;;
    --duration)
      DURATION_OPTION="--duration $2"
      shift
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

echo "🚀 Running auth service load tests..."
echo "📊 Test parameters:"
echo "   - Script: auth-service-load-test.js"
if [ ! -z "$VUS_OPTION" ]; then
  echo "   - VUs: $(echo $VUS_OPTION | cut -d ' ' -f2)"
else
  echo "   - VUs: Using stages from script"
fi
if [ ! -z "$DURATION_OPTION" ]; then
  echo "   - Duration: $(echo $DURATION_OPTION | cut -d ' ' -f2)"
else
  echo "   - Duration: Using stages from script"
fi
echo ""
echo "⏳ Starting test..."

# Execute the k6 test
k6 run $VUS_OPTION $DURATION_OPTION $OUTPUT_OPTIONS scripts/auth-service-load-test.js

# Check if HTML report was generated
if [ -f "summary.html" ]; then
  echo "📄 HTML report generated at: summary.html"
fi

echo "✅ Load test completed!"