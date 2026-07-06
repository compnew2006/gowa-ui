# Whatomate

Modern, open-source WhatsApp Business Platform. Support for both **WhatsApp Cloud API (Meta)** and **WhatsApp Web Protocol (whatsmeow)**. Single binary app.

![Dashboard](docs/public/images/dashboard-light.png#gh-light-mode-only)
![Dashboard](docs/public/images/dashboard-dark.png#gh-dark-mode-only)

## Features

- **Multi-tenant Architecture**
  Support multiple organizations with isolated data and configurations.

- **Dual Provider Support**
  Choose between **Meta's WhatsApp Cloud API** for official business integration or **Whatsmeow** (WhatsApp Web protocol) for QR-code based multi-instance support without API costs.

- **Granular Roles & Permissions**
  Customizable roles with fine-grained permissions. Create custom roles, assign specific permissions per resource (users, contacts, templates, chat, settings, and more), and control access with `read`, `write`, `delete`, and specialized actions such as `sync`, `execute`, `import`, `export`, `pickup`, `assign`, and `prefix`. Super admins can manage multiple organizations.
  In the permission matrix UI, `/settings/roles` appears under `Roles`, while `/settings/instances` uses the `WhatsApp Accounts` resource label.

- **Real-time Chat**
  Live messaging with WebSocket support for instant communication, featuring instance tags for easy identification.

- **Claim Audit System Messages**
  Each successful chat claim action writes a `chat_claimed` system message into the conversation timeline so teams can audit ownership changes directly in chat history.

- **Per-User Agent Name Prefix**
  Outgoing agent-authored text messages can be prefixed with the sender's full name (`Agent Name : message`) using each user's **Send Restrictions** setting (`prefix_agent_name`). Disable it per user to send plain text without an agent-name prefix.

- **Template Management**
  Create and manage message templates (Meta provider) or send regular messages (Whatsmeow).

- **Bulk Campaigns**
  Send campaigns to multiple contacts with retry support for failed messages.

- **Chatbot Automation**
  Keyword-based auto-replies, conversation flows with branching logic, and AI-powered responses (OpenAI, Anthropic, Google).

- **Canned Responses**
  Pre-defined quick replies with slash commands (`/shortcut`) and dynamic placeholders.

- **Conversation Notes**
  Add private notes to chats for internal team collaboration and historical context.

- **Message History Navigation**
  Effortlessly navigate through historical messages and search conversation history directly in the chat view.

- **Assigned Chat Reset**
  Reset assigned chats to unassigned status or reassign them to different agents to optimize workload.

- **Contact Management & Localization**
  Comprehensive contact management with custom fields, tagging, and multi-language support (including Spanish).

- **Analytics Dashboard**
  Track messages, engagement, and campaign performance.

<details>
<summary>View more screenshots</summary>

![Dashboard](docs/public/images/dashboard-light.png#gh-light-mode-only)
![Dashboard](docs/public/images/dashboard-dark.png#gh-dark-mode-only)
![Chatbot](docs/public/images/chatbot-light.png#gh-light-mode-only)
![Chatbot](docs/public/images/chatbot-dark.png#gh-dark-mode-only)
![Agent Analytics](docs/public/images/agent-analytics-light.png#gh-light-mode-only)
![Agent Analytics](docs/public/images/agent-analytics-dark.png#gh-dark-mode-only)
![Conversation Flow Builder](docs/public/images/conversation-flow-light.png#gh-light-mode-only)
![Conversation Flow Builder](docs/public/images/conversation-flow-dark.png#gh-dark-mode-only)
![Templates](docs/public/images/11-templates.png)
![Campaigns](docs/public/images/13-campaigns.png)

</details>

## Installation

### Docker

The latest image is available on Docker Hub at [`compnew2006/whatomate:latest`](https://hub.docker.com/r/compnew2006/whatomate)

```bash
# Download compose file and sample config
curl -LO https://raw.githubusercontent.com/compnew2006/whatomate/main/docker/docker-compose.yml
curl -LO https://raw.githubusercontent.com/compnew2006/whatomate/main/config.example.toml

# Copy and edit config
cp config.example.toml config.toml

# Generate JWT secret once and set it in config.toml (jwt.secret)
openssl rand -hex 32

# Generate encryption key and set it in config.toml (app.encryption_key)
openssl rand -hex 32

# Run services
docker compose up -d
```

Go to `http://localhost:8080` and login with `admin@admin.com` / `adminpassword12`.
Change `[default_admin].password` in `config.toml` before production use.
If the admin user already exists, reset it explicitly:
`./whatomate admin-reset-password -config config.toml -email admin@admin.com -password 'adminpassword12'`

---

### Binary

Download the [latest release](https://github.com/compnew2006/whatomate/releases) and extract the binary.

```bash
# Copy and edit config
cp config.example.toml config.toml

# Generate JWT secret once and set it in config.toml (jwt.secret)
openssl rand -hex 32

# Generate encryption key and set it in config.toml (app.encryption_key)
openssl rand -hex 32

# Run with migrations
./whatomate server -migrate
```

Go to `http://localhost:8080` and login with `admin@admin.com` / `adminpassword12`.
Change `[default_admin].password` in `config.toml` before production use.
If the admin user already exists, reset it explicitly:
`./whatomate admin-reset-password -config config.toml -email admin@admin.com -password 'adminpassword12'`

---

### Build from Source

```bash
git clone https://github.com/compnew2006/whatomate.git
cd whatomate

# Production build (single binary with embedded frontend)
make build-prod
./whatomate server -migrate
```

## Deployment

The binary produced by `make build-prod` is **standalone** and works solo. It includes the embedded frontend, so you do not need a separate frontend server.

To deploy in production:

1.  Copy the `whatomate` binary to your server.
2.  Provide a `config.toml` file.
3.  Ensure access to a PostgreSQL database and a Redis instance.
4.  Run the server: `./whatomate server -migrate`

See [configuration docs](https://github.com/compnew2006/whatomate/docs) for detailed setup options.

## CLI Usage

```bash
./whatomate server              # API + 1 worker (default)
./whatomate server -workers=0   # API only
./whatomate worker -workers=4   # Workers only (for scaling)
./whatomate version             # Show version
```

## Developers

The backend is written in Go ([Fastglue](https://github.com/zerodha/fastglue)) and the frontend is Vue.js 3 with shadcn-vue.

- If you are interested in contributing, please read [CONTRIBUTING.md](./CONTRIBUTING.md) first.

```bash
# Development setup (backend + frontend)
cd /Users/noiemany/Downloads/whatomate_GOWA/whatomate

# Node.js requirement for frontend (Vite 7): 20.19+ or 22.12+
cd frontend
nvm install
nvm use
cd ..

# 1) Start dependencies (Postgres + Redis)
docker compose -f docker/docker-compose.yml up -d db redis

# 2) Backend (Terminal 1, port 8080)
make run-migrate

# 3) Frontend (Terminal 2, port 3000)
cd frontend
npm install
npm run dev

# Optional: run backend + frontend together (after db/redis are up)
make dev

# Faster daily loop: backend hot reload + frontend hot reload
make dev-watch
```

Recommended workflow:

- Frontend-only changes: keep `cd frontend && npm run dev` running
- Backend-only changes: use `make backend-watch`
- Frontend + backend + model/schema changes: use `make dev-watch`
- Production-bundle verification only: use `make build-prod`

`make dev-watch` uses `air` for Go hot reload and reruns the backend with `-migrate`, so model/schema changes are picked up on restart. The first run auto-installs `air` if it is missing.

## License

See [LICENSE](LICENSE) for details.
