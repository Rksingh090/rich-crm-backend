# Rich CRM/ERP Deployment Guide

This guide explains how to deploy the **Rich CRM** and **Rich ERP** applications (along with the Go backend) to a Linux server.

## 1. Architecture Overview

The system consists of three main services running on specific ports:

| Service | Name | Port | Base Path |
| :--- | :--- | :--- | :--- |
| **Backend** | `rich-backend` | **42111** | `/api` |
| **CRM** | `rich-crm` | **42112** | `/crm` |
| **ERP** | `rich-erp` | **42113** | `/erp` |

An **Nginx** reverse proxy routes traffic from port **9001** to these services based on the URL path.

## 2. Server Setup (First Time Only)

These steps strictly need to be performed **once** to set up the infrastructure.

### Option A: Automated Setup (Recommended)
We have provided a script to automate the installation of Systemd services and Nginx configuration.

1. **Deploy the configuration files to the server:**
   Running the backend deployment script will also sync the deployment scripts to the server.
   ```bash
   ./backend/deployment/scripts/deploy-backend.sh
   # Note: Ensure SERVER_IP is valid in the script
   ```

2. **SSH into your server:**
   ```bash
   ssh root@<YOUR_SERVER_IP>
   ```

3. **Run the setup script:**
   ```bash
   cd /var/www/go-crm/backend/deployment/scripts
   chmod +x enable-rich-services.sh
   ./enable-rich-services.sh
   ```
   *This script will copy the systemd files, enable services, configur Nginx, and restart everything.*

### Option B: Manual Setup

1. **Copy Service Files:**
   Copy `backend/deployment/systemd/rich-*.service` to `/etc/systemd/system/`.
   ```bash
   # From local machine
   scp backend/deployment/systemd/rich-*.service root@<IP>:/etc/systemd/system/
   ```

2. **Enable Services:**
   ```bash
   # On Server
   systemctl daemon-reload
   systemctl enable --now rich-backend rich-crm rich-erp
   ```

3. **Configure Nginx:**
   Copy `backend/deployment/nginx/rich-app.conf` to `/etc/nginx/sites-available/` and link it.
   ```bash
   # From local machine
   scp backend/deployment/nginx/rich-app.conf root@<IP>:/etc/nginx/sites-available/rich-app.conf
   
   # On Server
   ln -s /etc/nginx/sites-available/rich-app.conf /etc/nginx/sites-enabled/
   rm /etc/nginx/sites-enabled/default  # Optional: Remove default
   nginx -t
   systemctl reload nginx
   ```

## 3. Routine Deployment

Use the provided scripts to deploy updates to individual components. Run these from your **local project root**.

### Deploy Backend
Builds the Go binary and restarts `rich-backend`.
```bash
./backend/deployment/scripts/deploy-backend.sh
```

### Deploy CRM
Syncs the `frontend/crm` code, builds it on the server, and restarts `rich-crm`.
```bash
./backend/deployment/scripts/deploy-crm.sh
```

### Deploy ERP
Syncs the `frontend/erp` code, builds it on the server, and restarts `rich-erp`.
```bash
./backend/deployment/scripts/deploy-erp.sh
```

## 4. Troubleshooting

- **Check Service Status:**
  ```bash
  systemctl status rich-backend
  systemctl status rich-crm
  systemctl status rich-erp
  ```
- **Check Logs:**
  ```bash
  journalctl -u rich-backend -f
  journalctl -u rich-crm -f
  ```
- **Nginx Errors:**
  ```bash
  tail -f /var/log/nginx/error.log
  ```
