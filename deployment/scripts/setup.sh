#!/bin/bash

# setup.sh - Setup and Run Go-CRM

set -e # Exit on error

GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${GREEN}[*] Setting up Go-CRM...${NC}"

# 1. Backend Setup
echo -e "${GREEN}[*] Installing Backend Dependencies...${NC}"
go mod download
if ! command -v air &> /dev/null; then
    echo "Air not found, installing..."
    go install github.com/air-verse/air@latest
fi

echo -e "${GREEN}[*] Seeding Database...${NC}"
make seed || echo -e "${GREEN}[!] Seeding failed or already done, continuing...${NC}"

# 2. Frontend Setup
echo -e "${GREEN}[*] Installing Frontend Dependencies...${NC}"
cd ui
yarn install
cd ..

# 3. Environment Check
if [ ! -f .env ]; then
    echo -e "${GREEN}[!] .env file not found. Please create one.${NC}"
    # touch .env # Optional: create empty one
fi

echo -e "${GREEN}[*] Setup Complete! Starting services...${NC}"

# 4. Run Services
# Trap SIGINT to kill background processes when script is exiting
trap 'kill $(jobs -p)' SIGINT

echo -e "${GREEN}[*] Starting Backend (Air)...${NC}"
make run & # Run backend in background
BACKEND_PID=$!

echo -e "${GREEN}[*] Starting Frontend (Next.js)...${NC}"
cd ui && yarn dev

# Wait for background process
wait $BACKEND_PID
