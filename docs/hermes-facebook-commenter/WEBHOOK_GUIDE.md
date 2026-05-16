# Facebook Webhook Setup Guide

## Quick Setup (3 steps)

### 1️⃣ Make your local server public

**Option A: ngrok (Recommended)**
```bash
# Install ngrok
curl -s https://ngrok-agent.s3.amazonaws.com/ngrok.asc | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null
echo 'deb https://ngrok-agent.s3.amazonaws.com buster main' | sudo tee /etc/apt/sources.list.d/ngrok.list
sudo apt update && sudo apt install ngrok

# Start tunnel
ngrok http 5000
```

**Option B: localtunnel**
```bash
npm install -g localtunnel
lt --port 5000
```

**Option C: Deploy** (Railway, Render, AWS, etc.)

### 2️⃣ Configure Meta Webhook

1. Go to https://developers.facebook.com/apps
2. Select your app → Products → Webhooks
3. Click "Add Callback URL"
4. Enter:
   - **Callback URL**: `https://your-url.ngrok.io/webhook`
   - **Verify Token**: `hermes_facebook_verify` (or run the setup script for a random token)
5. Click "Verify and Save"
6. Subscribe to: `comments`, `feed`

### 3️⃣ Start the Webhook Server

```bash
cd ~/.hermes/plugins
python3 facebook_webhook.py
```

Or run the automated setup:
```bash
~/.hermes/plugins/webhook_setup.sh
```

## Subscribe Your Page

After setting up the webhook, subscribe your page to receive events:

```bash
curl -X POST \
  "https://graph.facebook.com/v19.0/$FB_PAGE_ID/subscribed_apps" \
  -d "access_token=$FB_PAGE_ACCESS_TOKEN" \
  -d "subscribed_fields=comments,feed"
```

## Test It

Post a comment on any Facebook page post. You should see:
```
📩 New comment from John Doe: Great post!
✓ Forwarded to Hermes
```

## Webhook URL Format

Your final webhook URL will be:
```
https://your-public-domain.com/webhook
```

Examples:
- ngrok: `https://abc123.ngrok.io/webhook`
- localtunnel: `https://your-name.loca.lt/webhook`
- custom: `https://your-server.com/webhook`

## Environment Variables

```bash
# Required
export FB_PAGE_ACCESS_TOKEN="your_token"
export FB_PAGE_ID="your_page_id"

# Optional (custom verification token)
export FB_WEBHOOK_VERIFY_TOKEN="your_custom_token"

# Optional (Hermes API endpoint)
export HERMES_API_URL="http://localhost:8080"
```

## Production Deployment

For production, deploy the webhook server to:
- **Railway**: `railway up`
- **Render**: Connect GitHub repo
- **AWS/GCP/Azure**: Any cloud service

Your webhook URL will be provided by the platform.
