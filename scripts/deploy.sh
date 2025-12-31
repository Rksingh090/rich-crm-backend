#!/bin/bash

# Configuration
SERVER_USER="root"
SERVER_IP="93.127.172.45"
DEPLOY_PATH="/var/www/go-crm"
BINARY_NAME="api"
UI_PATH="ui"

# --- Colors ---
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting deployment...${NC}"

# 0. Preparation
echo -e "${GREEN}Preparing server directory...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ${DEPLOY_PATH}"

# 1. Build Backend Locally (Ensuring Linux AMD64 architecture)
echo -e "${GREEN}Building backend for Linux AMD64...${NC}"
GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME} cmd/api/main.go

# 2. Transfer Source Code and Binary
echo -e "${GREEN}Transferring files to server...${NC}"
# Use rsync to efficiently transfer files
rsync -avz --progress \
    --exclude 'node_modules' \
    --exclude '.next' \
    --exclude '.git' \
    --exclude 'ui/node_modules' \
    --exclude 'ui/.next' \
    --exclude 'api' \
    ./ ${SERVER_USER}@${SERVER_IP}:${DEPLOY_PATH}/

# 3. Build Frontend on Server
echo -e "${GREEN}Building frontend on server...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cd ${DEPLOY_PATH}/${UI_PATH} && \
    if [ -f \"yarn.lock\" ]; then yarn install --ignore-engines && yarn build; else npm install --engine-strict=false && npm run build; fi && \
    rm -rf .next/standalone/public .next/standalone/.next/static && \
    cp -r public .next/standalone/ && \
    mkdir -p .next/standalone/.next && \
    cp -r .next/static .next/standalone/.next/"

# 4. Restart Services
echo -e "${GREEN}Restarting services...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "systemctl restart go-crm go-crm-ui"

echo -e "${BLUE}Deployment completed successfully!${NC}"
