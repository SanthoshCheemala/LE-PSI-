#!/bin/bash

# LE-PSI Scalability Test Runner
# This script sets up and runs comprehensive scalability tests

set -e

echo "================================================="
echo "  LE-PSI SCALABILITY TEST SETUP"
echo "================================================="
echo ""

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    echo "Error: main.go not found. Please run from scalability_tests directory"
    exit 1
fi

# Initialize Go module if needed
echo "📦 Setting up Go module..."
if [ ! -f "go.mod" ]; then
    go mod init scalability_tests
fi

# Add replace directive for local PSI package
echo "🔗 Linking PSI library..."
go mod edit -replace github.com/SanthoshCheemala/PSI=..

# Install dependencies
echo "⬇️  Installing dependencies..."
go get github.com/mattn/go-sqlite3
go mod tidy

echo ""
echo "✅ Setup complete!"
echo ""

# Create results directory
mkdir -p scalability_results

# Ask user what to do
echo "What would you like to do?"
echo ""
echo "1) Run all scalability tests (recommended - uses real database)"
echo "2) Run database integration test only (50K records)"
echo "3) Build executable for later use"
echo "4) View existing results"
echo ""
read -p "Enter choice [1-4]: " choice

case $choice in
    1)
        echo ""
        echo "🚀 Running ALL scalability tests..."
        echo "This may take 30-60 minutes depending on your hardware."
        echo "All tests use REAL data from ../data/transactions.db"
        echo ""
        read -p "Continue? (y/n): " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            go run main.go
            
            # Open the latest HTML report
            latest_html=$(ls -t scalability_results/*.html 2>/dev/null | head -1)
            if [ -n "$latest_html" ]; then
                echo ""
                echo "✅ Tests complete! Opening report..."
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    open "$latest_html"
                elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
                    xdg-open "$latest_html"
                fi
            fi
        fi
        ;;
    
    2)
        echo ""
        echo "🗄️  Running database integration test..."
        
        # Check if database exists
        if [ ! -f "../data/transactions.db" ]; then
            echo "❌ ERROR: Database not found at ../data/transactions.db"
            echo "   Cannot run tests without real data!"
            exit 1
        else
            echo "✅ Found database: ../data/transactions.db"
        fi
        
        echo ""
        read -p "Continue? (y/n): " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            echo "Running database integration test (50K records)..."
            go run main.go
        fi
        ;;
    
    3)
        echo ""
        echo "🔨 Building executable..."
        go build -o scalability_test main.go
        echo "✅ Built: ./scalability_test"
        echo ""
        echo "Run with: ./scalability_test"
        ;;
    
    4)
        echo ""
        echo "📊 Existing test results:"
        echo ""
        
        if [ -d "scalability_results" ]; then
            ls -lh scalability_results/
            echo ""
            
            # Show latest HTML report
            latest_html=$(ls -t scalability_results/*.html 2>/dev/null | head -1)
            if [ -n "$latest_html" ]; then
                echo "Latest report: $latest_html"
                read -p "Open it? (y/n): " open_it
                if [ "$open_it" = "y" ] || [ "$open_it" = "Y" ]; then
                    if [[ "$OSTYPE" == "darwin"* ]]; then
                        open "$latest_html"
                    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
                        xdg-open "$latest_html"
                    fi
                fi
            else
                echo "No HTML reports found. Run tests first."
            fi
        else
            echo "No results directory found. Run tests first."
        fi
        ;;
    
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "================================================="
echo "  Done!"
echo "================================================="
