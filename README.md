<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>

# Gowa-UI

Modern, open-source WhatsApp Business Platform. Single binary app.

![Dashboard](docs/public/images/dashboard-light.png#gh-light-mode-only)
![Dashboard](docs/public/images/dashboard-dark.png#gh-dark-mode-only)

## Features

- **Multi-tenant Architecture**
  Support multiple organizations with isolated data and configurations.

- **Granular Roles & Permissions**
  Customizable roles with fine-grained permissions. Create custom roles, assign specific permissions per resource (users, contacts, templates, etc.), and control access at the action level (read, create, update, delete). Super admins can manage multiple organizations.

- **WhatsApp Business API Integration**
  Connect via the WhatsApp Business API for messaging.

- **Real-time Chat**
  Live messaging with WebSocket support for instant communication.

- **Template Management**
  Create and manage message templates approved by Meta.

- **Bulk Campaigns**
  Send campaigns to multiple contacts with retry support for failed messages.

- **WhatsApp Account Management**
  Connect and manage multiple WhatsApp accounts via GOWA (WhatsApp Web protocol) with user-specific visibility controls and granular device management.

- **Canned Responses**
  Pre-defined quick replies with slash commands (`/shortcut`) and dynamic placeholders.

- **Analytics Dashboard**
  Track messages, engagement, and campaign performance.

<details>
<summary>View more screenshots</summary>

![Dashboard](docs/public/images/dashboard-light.png#gh-light-mode-only)
![Dashboard](docs/public/images/dashboard-dark.png#gh-dark-mode-only)
![Agent Analytics](docs/public/images/agent-analytics-light.png#gh-light-mode-only)
![Agent Analytics](docs/public/images/agent-analytics-dark.png#gh-dark-mode-only)
![Templates](docs/public/images/11-templates.png)
![Campaigns](docs/public/images/13-campaigns.png)

</details>

## Installation

### Docker

The latest image is available on Docker Hub at [`compnew2006/gowa-ui:latest`](https://hub.docker.com/r/compnew2006/gowa-ui)

```bash
# Download compose file, sample config, and env file
curl -LO https://raw.githubusercontent.com/compnew2006/gowa-ui/main/docker/docker-compose.yml
curl -LO https://raw.githubusercontent.com/compnew2006/gowa-ui/main/config.example.toml
curl -L https://raw.githubusercontent.com/compnew2006/gowa-ui/main/docker/.env.example -o .env

# Copy and edit config
cp config.example.toml config.toml
# Edit .env to set PostgreSQL credentials and timezone

# Run services
docker compose up -d
```

Go to `http://localhost:8080` and login with `admin@admin.com` / `admin`

__________________

### Binary

Download the [latest release](https://github.com/compnew2006/gowa-ui/releases) and extract the binary.

```bash
# Copy and edit config
cp config.example.toml config.toml

# Run with migrations
./gowa-ui server -migrate
```

Go to `http://localhost:8080` and login with `admin@admin.com` / `admin`

__________________

### Build from Source

```bash
git clone https://github.com/compnew2006/gowa-ui.git
cd gowa-ui

# Production build (single binary with embedded frontend)
make build-prod
./gowa-ui server -migrate
```

See [configuration docs](https://compnew2006.github.io/gowa-ui/getting-started/configuration/) for detailed setup options.

## CLI Usage

```bash
./gowa-ui server              # API + 1 worker (default)
./gowa-ui server -workers=0   # API only
./gowa-ui worker -workers=4   # Workers only (for scaling)
./gowa-ui version             # Show version
```

## Developers

The backend is written in Go ([Fastglue](https://github.com/zerodha/fastglue)) and the frontend is Vue.js 3 with shadcn-vue.
- If you are interested in contributing, please read [CONTRIBUTING.md](./CONTRIBUTING.md) first.

```bash
# Development setup
make run-migrate    # Backend (port 8080)
cd frontend && npm run dev   # Frontend (port 3000)
```

## License

See [LICENSE](LICENSE) for details.
