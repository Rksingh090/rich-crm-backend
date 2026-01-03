#!/bin/bash

# enable-rich-services.sh
# Run this script ON THE REMOTE SERVER (as root) to enable Rich CRM/ERP services and Nginx config.
# Assumes you have already copied the systemd files and nginx config to the server, 
# OR you are running this from the deployed /var/www/go-crm/backend/scripts directory.

# --- Configuration ---
APP_DIR="/var/www/go-crm/deployment" # Adjust if your path is different
NGINX_AVAILABLE="/etc/nginx/sites-available"
NGINX_ENABLED="/etc/nginx/sites-enabled"
SYSTEMD_DIR="/etc/systemd/system"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}[*] Starting Rich Services Activation...${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
  echo -e "${RED}[!] Please run as root (sudo).${NC}"
  exit 1
fi

# 1. Install Systemd Services
echo -e "${GREEN}[*] Installing Systemd Services...${NC}"

# Check if source files exist in standard deployment location
if [ -f "$APP_DIR/systemd/rich-backend.service" ]; then
    cp "$APP_DIR/systemd/rich-backend.service" "$SYSTEMD_DIR/"
    cp "$APP_DIR/systemd/rich-crm.service" "$SYSTEMD_DIR/"
    cp "$APP_DIR/systemd/rich-erp.service" "$SYSTEMD_DIR/"
    cp "$APP_DIR/systemd/rich-web.service" "$SYSTEMD_DIR/"
else
    echo -e "${RED}[!] Service files not found in $APP_DIR/systemd/. Please verify deployment.${NC}"
    # Try to find in current dir as fallback
    if [ -f "rich-backend.service" ]; then
        echo -e "${BLUE}[i] Found services in current directory. Using them.${NC}"
        cp rich-*.service "$SYSTEMD_DIR/"
    else
        exit 1
    fi
fi

systemctl daemon-reload
systemctl enable rich-backend rich-crm rich-erp rich-web
systemctl restart rich-backend rich-crm rich-erp rich-web
echo -e "${GREEN}[+] Services enabled and restarted.${NC}"

# 2. Configure Nginx
echo -e "${GREEN}[*] Configuring Nginx...${NC}"

if [ -f "$APP_DIR/nginx/rich-app.conf" ]; then
    cp "$APP_DIR/nginx/rich-app.conf" "$NGINX_AVAILABLE/rich-app.conf"
elif [ -f "rich-app.conf" ]; then
    cp rich-app.conf "$NGINX_AVAILABLE/rich-app.conf"
else
    echo -e "${RED}[!] Nginx config rich-app.conf not found.${NC}"
    exit 1
fi

# Link
ln -sf "$NGINX_AVAILABLE/rich-app.conf" "$NGINX_ENABLED/rich-app.conf"

# Remove default if exists
if [ -f "$NGINX_ENABLED/default" ]; then
    echo -e "${BLUE}[i] Removing default Nginx config...${NC}"
    rm "$NGINX_ENABLED/default"
fi

# Test and Reload
echo -e "${BLUE}[*] Testing Nginx configuration...${NC}"
nginx -t

if [ $? -eq 0 ]; then
    systemctl reload nginx
    echo -e "${GREEN}[+] Nginx reloaded successfully!${NC}"
    echo -e "${GREEN}[SUCCESS] Deployment of Rich Services key infrastructure is complete.${NC}"
else
    echo -e "${RED}[!] Nginx configuration test failed. Previous config kept.${NC}"
    exit 1
fi
