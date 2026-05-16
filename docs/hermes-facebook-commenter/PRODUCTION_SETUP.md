# 🎯 Facebook Webhook Setup for ofuqalmadenah.com

## Quick Setup (One Command)

```bash
sudo ~/.hermes/plugins/setup_production_webhook.sh
```

This will automatically:
- ✅ Install all dependencies (Python, nginx, certbot)
- ✅ Configure webhook server
- ✅ Setup SSL certificate
- ✅ Configure nginx reverse proxy
- ✅ Create systemd service (auto-restart)
- ✅ Generate secure verify token

## What You'll Need

1. **Static IP** - Already have ✓
2. **Domain** - `*.ofuqalmadenah.com` - Already have ✓
3. **Subdomain name** - Choose one (e.g., `fbwebhook`, `webhook`, `fb`)
4. **Facebook Page ID** - From Meta Developers
5. **Facebook Page Access Token** - From Meta Developers

## Manual DNS Configuration

Before running the script, add this DNS record:

**In your domain control panel** (where you bought ofuqalmadenah.com):

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | fbwebhook | YOUR_STATIC_IP | 300 |

Your final webhook URL will be: `https://fbwebhook.ofuqalmadenah.com/webhook`

## What the Script Does

### 1. System Setup
```bash
- Installs Python 3, pip, nginx, certbot, gunicorn
- Creates virtual environment at /opt/hermes-webhook
- Installs Flask and dependencies
```

### 2. Webhook Server
```bash
- Copies webhook server to /opt/hermes-webhook
- Creates .env file with your tokens
- Generates random verify token
```

### 3. nginx Configuration
```bash
- Creates reverse proxy to localhost:5000
- Configures /webhook endpoint
- Enables site
```

### 4. SSL Certificate
```bash
- Runs certbot for Let's Encrypt SSL
- Auto-configures HTTPS redirect
- Sets up auto-renewal
```

### 5. System Service
```bash
- Creates systemd service
- Enables auto-start on boot
- Auto-restart on failure
```

## Your Webhook URLs

After setup:

```
Webhook:  https://fbwebhook.ofuqalmadenah.com/webhook
Health:   https://fbwebhook.ofuqalmadenah.com/health
Verify Token: (shown during setup)
```

## Configure Meta Developers

1. Go to https://developers.facebook.com/apps
2. Select your app → Products → Webhooks
3. Add Callback URL:
   - **Callback URL**: `https://fbwebhook.ofuqalmadenah.com/webhook`
   - **Verify Token**: (from setup output)
4. Subscribe to: `comments`, `feed`

## Subscribe Your Page

```bash
curl -X POST \
  "https://graph.facebook.com/v19.0/YOUR_PAGE_ID/subscribed_apps" \
  -d "access_token=YOUR_PAGE_ACCESS_TOKEN" \
  -d "subscribed_fields=comments,feed"
```

## Test Your Setup

```bash
# Check service status
sudo systemctl status hermes-facebook-webhook

# View logs
sudo journalctl -u hermes-facebook-webhook -f

# Check nginx
sudo nginx -t

# Test health endpoint
curl https://fbwebhook.ofuqalmadenah.com/health
```

## Service Management

```bash
# Start service
sudo systemctl start hermes-facebook-webhook

# Stop service
sudo systemctl stop hermes-facebook-webhook

# Restart service
sudo systemctl restart hermes-facebook-webhook

# View logs
sudo journalctl -u hermes-facebook-webhook -f
tail -f /var/log/hermes/facebook_webhook.log
```

## Firewall Configuration

If you have a firewall, allow:

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 22/tcp
sudo ufw enable
```

## Troubleshooting

### SSL Certificate Issues
```bash
# Renew manually
sudo certbot renew

# Force renewal
sudo certbot renew --force-renewal
```

### Service Not Starting
```bash
# Check logs
sudo journalctl -u hermes-facebook-webhook -n 50

# Check if port 5000 is in use
sudo lsof -i :5000
```

### DNS Not Propagating
```bash
# Check DNS
dig fbwebhook.ofuqalmadenah.com

# Or
nslookup fbwebhook.ofuqalmadenah.com
```

## File Locations

```
Webhook server:    /opt/hermes-webhook/facebook_webhook_gunicorn.py
Environment:       /opt/hermes-webhook/.env
nginx config:      /etc/nginx/sites-available/fbwebhook.ofuqalmadenah.com
Service file:      /etc/systemd/system/hermes-facebook-webhook.service
Logs:              /var/log/hermes/facebook_webhook.log
Event queue:       ~/.hermes/webhook_queue/
```

## Security Notes

- ✅ HTTPS enforced with SSL
- ✅ Runs as non-root user (www-data)
- ✅ Random verify token generated
- ✅ Only exposes /webhook and /health endpoints
- ✅ nginx reverse proxy for security
- ✅ Auto-restart on failure

## Multiple Subdomains

You can run multiple webhooks on different subdomains:

```
fbwebhook.ofuqalmadenah.com  - Main webhook
fbtest.ofuqalmadenah.com     - Testing
fbbackup.ofuqalmadenah.com   - Backup
```

Just run the setup script multiple times with different subdomains.
