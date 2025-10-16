#!/bin/bash

# Trustroots KPI Dashboard - Python Server Startup Script
# This script starts a Python HTTP server to serve the static dashboard files

set -e

# Configuration
PORT=8303
DIRECTORY="public"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Trustroots KPI Dashboard Server${NC}"
echo -e "${YELLOW}Port: ${PORT}${NC}"
echo -e "${YELLOW}Directory: ${DIRECTORY}${NC}"
echo ""

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}❌ Python 3 is not installed or not in PATH${NC}"
    exit 1
fi

# Check if the public directory exists
if [ ! -d "$DIRECTORY" ]; then
    echo -e "${RED}❌ Directory '${DIRECTORY}' does not exist${NC}"
    exit 1
fi

# Check if kpi.json exists
if [ ! -f "${DIRECTORY}/kpi.json" ]; then
    echo -e "${YELLOW}⚠️  Warning: kpi.json not found in ${DIRECTORY}/ directory${NC}"
    echo -e "${YELLOW}   You may need to run the Go KPI service first to generate data${NC}"
    echo ""
fi

# Check if port is already in use
if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Port ${PORT} is already in use${NC}"
    echo -e "${YELLOW}   Attempting to kill existing process...${NC}"
    pkill -f "python3 -m http.server ${PORT}" || true
    sleep 2
fi

echo -e "${GREEN}✅ Starting server...${NC}"
echo -e "${BLUE}📊 Dashboard will be available at: http://localhost:${PORT}${NC}"
echo -e "${BLUE}📁 Serving files from: $(pwd)/${DIRECTORY}${NC}"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
echo ""

# Start the Python HTTP server
python3 -m http.server $PORT --directory $DIRECTORY
