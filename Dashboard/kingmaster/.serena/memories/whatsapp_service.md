# WhatsApp Service (Node.js)

**Location**: Project root (`server.js` + `sessionManager.js`).
**Runtime**: Node.js with Express 4.18.
**Library**: `@wppconnect-team/wppconnect ^1.30`.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/` | Service info and endpoint listing |
| POST | `/session/start` | Create WhatsApp session, returns QR |
| GET | `/session/:name/qr` | Get QR code for existing session |
| GET | `/session/:name/status` | Check session status |
| GET | `/sessions` | List all active sessions |
| DELETE | `/session/:name` | Close a session |
| POST | `/message/send` | Send a message via session |

## Config (env vars)
- `PORT` — default 3000
- `WPP_API_KEY` — if set, all requests require `x-api-key` header
- `WPP_ALLOWED_ORIGINS` — comma-separated CORS origins

## Validation
- Session names: `/^[A-Za-z0-9_-]{1,64}$/`
- Phone numbers: `/^[0-9+@.\-_\s]{5,40}$/`
- Messages: non-empty string, max 4096 chars

## Binding
Listens on `127.0.0.1` only (not externally accessible). PHP backend calls it locally.
