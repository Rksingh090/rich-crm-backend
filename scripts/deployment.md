# Go-CRM Deployment Guide

This guide explains how to deploy the Go-CRM project to a Linux server using the provided scripts and templates.

## 1. Server Prerequisites

Ensure your server has the following installed:
- **MongoDB**: The backend requires a running MongoDB instance.
- **Nginx**: For reverse proxy and SSL (recommended).
- **Node.js**: If running the Next.js frontend as a standalone server.

## 2. Local Preparation

Before deploying, ensure you have configured your server details in `scripts/deploy.sh`:

```bash
SERVER_USER="your_user"
SERVER_IP="your_server_ip"
DEPLOY_PATH="/var/www/go-crm"
```

## 3. Server Setup

### Systemd Services
1. Copy `go-crm.service.template` to `/etc/systemd/system/go-crm.service` and `go-crm-ui.service.template` to `/etc/systemd/system/go-crm-ui.service` on your server.
2. Edit the service files to match your directory and environment variables.
3. Reload systemd and start the services:
```bash
sudo systemctl daemon-reload
sudo systemctl enable go-crm go-crm-ui
sudo systemctl start go-crm go-crm-ui
```

### Nginx Configuration
1. Copy `nginx.conf.template` to `/etc/nginx/sites-available/go-crm`.
2. Link it to `sites-enabled`: `sudo ln -s /etc/nginx/sites-available/go-crm /etc/nginx/sites-enabled/`.
3. Test Nginx config: `sudo nginx -t`.
4. Restart Nginx: `sudo systemctl restart nginx`.

## 4. Run Deployment

Execute the deployment script from your local machine:

```bash
chmod +x scripts/deploy.sh
./scripts/deploy.sh
```

This script will:
- Cross-compile the Go backend for Linux.
- Build the Next.js frontend.
- Transfer files via `scp`.
- Restart the backend service on the server.

## 5. Security Notes
- Use **SSH keys** for authentication to avoid password prompts during deployment.
- Configure a **Firewall** (UFW) to only allow ports 80, 443, and 22.
- Set up **SSL** using Let's Encrypt (`certbot`).
