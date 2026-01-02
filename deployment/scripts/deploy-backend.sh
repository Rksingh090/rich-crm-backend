#!/bin/bash

# Configuration
SERVER_USER="root"
SERVER_IP="93.127.172.45"
DEPLOY_PATH="/var/www/go-crm"
BINARY_NAME="api"

# --- Colors ---
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting BACKEND deployment...${NC}"

# Ensure we are in project root (check for backend folder)
if [ ! -d "backend" ]; then
    echo -e "${BLUE}Please run this script from the project root.${NC}"
    exit 1
fi

# 1. Build Backend Locally
echo -e "${GREEN}Building backend for Linux AMD64...${NC}"
cd backend
GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME} cmd/api/main.go
if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi
cd ..

# 2. Transfer Backend Files
echo -e "${GREEN}Transferring backend files to server...${NC}"
# Sync contents of backend/ to DEPLOY_PATH
rsync -avz --progress \
    --exclude 'node_modules' \
    --exclude '.git' \
    --exclude 'ui' \
    backend/ ${SERVER_USER}@${SERVER_IP}:${DEPLOY_PATH}/

# 3. Restart Backend Service
echo -e "${GREEN}Restarting backend service...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "systemctl restart rich-backend"

echo -e "${BLUE}Backend deployment completed successfully!${NC}"
