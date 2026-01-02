#!/bin/bash

# Configuration
SERVER_USER="root"
SERVER_IP="93.127.172.45"
DEPLOY_PATH="/var/www/go-crm"
UI_PATH="ui"

# --- Colors ---
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting CRM (Frontend) deployment...${NC}"

# Ensure we are in project root
if [ ! -d "frontend/crm" ]; then
    echo -e "${BLUE}Please run this script from the project root.${NC}"
    exit 1
fi

# 1. Prepare Server Directory
echo -e "${GREEN}Preparing server directory...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ${DEPLOY_PATH}/${UI_PATH}"

# 2. Transfer CRM Files
echo -e "${GREEN}Transferring CRM files to server...${NC}"
rsync -avz --progress \
    --exclude 'node_modules' \
    --exclude '.next' \
    --exclude '.git' \
    frontend/crm/ ${SERVER_USER}@${SERVER_IP}:${DEPLOY_PATH}/${UI_PATH}/

# 3. Build CRM on Server
echo -e "${GREEN}Building CRM on server...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "cd ${DEPLOY_PATH}/${UI_PATH} && \
    if [ -f \"yarn.lock\" ]; then yarn install --ignore-engines && yarn build; else npm install --engine-strict=false && npm run build; fi && \
    rm -rf .next/standalone/public .next/standalone/.next/static && \
    if [ -d "public" ]; then cp -r public .next/standalone/; fi && \
    mkdir -p .next/standalone/.next && \
    cp -r .next/static .next/standalone/.next/"

# 4. Restart CRM Service
echo -e "${GREEN}Restarting CRM service...${NC}"
ssh ${SERVER_USER}@${SERVER_IP} "systemctl restart rich-crm"

echo -e "${BLUE}CRM deployment completed successfully!${NC}"
