#!/bin/bash

# Trustroots KPI Dashboard - Development Script
# This script helps with development tasks like starting servers and running the KPI service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SERVER_PORT=8303
SERVER_DIRECTORY="public"

show_help() {
    echo -e "${BLUE}Trustroots KPI Dashboard - Development Script${NC}"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  server     Start Python HTTP server (default)"
    echo "  kpi        Run Go KPI service once to generate data"
    echo "  kpi-dev    Run Go KPI service with local MongoDB"
    echo "  build      Build the Go KPI service"
    echo "  clean      Clean up generated files"
    echo "  help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 server     # Start the dashboard server"
    echo "  $0 kpi-dev    # Generate KPI data with local MongoDB"
    echo "  $0 build      # Build the Go service"
}

start_server() {
    echo -e "${BLUE}🚀 Starting Trustroots KPI Dashboard Server${NC}"
    echo -e "${YELLOW}Port: ${SERVER_PORT}${NC}"
    echo -e "${YELLOW}Directory: ${SERVER_DIRECTORY}${NC}"
    echo ""

    # Check if Python 3 is available
    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}❌ Python 3 is not installed or not in PATH${NC}"
        exit 1
    fi

    # Check if the public directory exists
    if [ ! -d "$SERVER_DIRECTORY" ]; then
        echo -e "${RED}❌ Directory '${SERVER_DIRECTORY}' does not exist${NC}"
        exit 1
    fi

    # Check if kpi.json exists
    if [ ! -f "${SERVER_DIRECTORY}/kpi.json" ]; then
        echo -e "${YELLOW}⚠️  Warning: kpi.json not found in ${SERVER_DIRECTORY}/ directory${NC}"
        echo -e "${YELLOW}   You may need to run: $0 kpi-dev${NC}"
        echo ""
    fi

    # Check if port is already in use
    if lsof -Pi :$SERVER_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Port ${SERVER_PORT} is already in use${NC}"
        echo -e "${YELLOW}   Attempting to kill existing process...${NC}"
        pkill -f "python3 -m http.server ${SERVER_PORT}" || true
        sleep 2
    fi

    echo -e "${GREEN}✅ Starting server...${NC}"
    echo -e "${BLUE}📊 Dashboard will be available at: http://localhost:${SERVER_PORT}${NC}"
    echo -e "${BLUE}📁 Serving files from: $(pwd)/${SERVER_DIRECTORY}${NC}"
    echo ""
    echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
    echo ""

    # Start the Python HTTP server
    python3 -m http.server $SERVER_PORT --directory $SERVER_DIRECTORY
}

run_kpi() {
    echo -e "${BLUE}📊 Running KPI Data Collection${NC}"
    echo ""

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
        exit 1
    fi

    # Check if main.go exists
    if [ ! -f "main.go" ]; then
        echo -e "${RED}❌ main.go not found in current directory${NC}"
        exit 1
    fi

    echo -e "${YELLOW}⚠️  Note: This will try to connect to the default MongoDB URI${NC}"
    echo -e "${YELLOW}   If you need to specify a different URI, set MONGO_URI environment variable${NC}"
    echo ""

    # Run the Go KPI service once
    go run main.go --once
}

run_kpi_dev() {
    echo -e "${BLUE}📊 Running KPI Data Collection (Development Mode)${NC}"
    echo ""

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
        exit 1
    fi

    # Check if main.go exists
    if [ ! -f "main.go" ]; then
        echo -e "${RED}❌ main.go not found in current directory${NC}"
        exit 1
    fi

    echo -e "${YELLOW}⚠️  Using localhost MongoDB for development${NC}"
    echo ""

    # Run with localhost MongoDB
    MONGO_URI=mongodb://localhost:27017 go run main.go --once
}

build_kpi() {
    echo -e "${BLUE}🔨 Building KPI Service${NC}"
    echo ""

    # Check if Go is available
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed or not in PATH${NC}"
        exit 1
    fi

    # Build the service
    go build -o kpi-service main.go

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Build successful! Binary created: kpi-service${NC}"
    else
        echo -e "${RED}❌ Build failed${NC}"
        exit 1
    fi
}

clean_files() {
    echo -e "${BLUE}🧹 Cleaning up generated files${NC}"
    echo ""

    # Remove built binary
    if [ -f "kpi-service" ]; then
        rm kpi-service
        echo -e "${GREEN}✅ Removed kpi-service binary${NC}"
    fi

    # Optionally remove kpi.json (commented out for safety)
    # if [ -f "public/kpi.json" ]; then
    #     rm public/kpi.json
    #     echo -e "${GREEN}✅ Removed kpi.json${NC}"
    # fi

    echo -e "${GREEN}✅ Cleanup completed${NC}"
}

# Main script logic
case "${1:-server}" in
    "server")
        start_server
        ;;
    "kpi")
        run_kpi
        ;;
    "kpi-dev")
        run_kpi_dev
        ;;
    "build")
        build_kpi
        ;;
    "clean")
        clean_files
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        echo -e "${RED}❌ Unknown command: $1${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
