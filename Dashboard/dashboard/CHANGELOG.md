# Changelog

All notable changes to this project will be documented in this file.

## [2026-05-23]

### Added
- **Frontend Authentication Navigation Guard:** Integrated route checking in `AppShell` (`app-shell.tsx`) to isolate `/login` and `/register` public pages and enforce automatic redirection of unauthenticated users to `/login` with a sleek glassmorphism loading animation.
- **Sidebar Logout and User Profile Card:** Enhanced `Sidebar` (`sidebar.tsx`) to display the current logged-in user's name and role in a premium visual container and added a secure "تسجيل الخروج" (Logout) button to clear JWT session states and redirect securely.
- **Meta WhatsApp Cloud API Integration:** Implemented official Meta WhatsApp Cloud API Service (`whatsapp_cloud_api.py`) with direct database row querying to bypass access-token serialization masking.
- **WhatsApp Tasks & Scheduled Posts Fallback:** Rewrote scheduled posts and conversation reply pipelines (`tasks.py`) to leverage the Meta Cloud API for sending messages, falling back automatically and gracefully to the local WhatsApp bridge `/send` and `/broadcast` endpoints in case Meta credentials are unconfigured or fail.
- **WhatsApp Cloud API Settings UI:** Added custom, secure configuration fields for the official Meta Cloud API in the System Settings page (`settings/page.tsx`) with full Arabic localization.

### Fixed
- **API Server Startup (`api-server/start.sh`):** Fixed hardcoded `/home/runner` path to relative local directory path and updated the default port from `8080` to `8000` to match the default Next.js proxy settings.
- **Database Schema Mismatch:** Resolved `UndefinedColumnError` on the PostgreSQL database by adding missing columns to the pre-existing tables (`whatsapp_id` to `customers`; `whatsapp_notification_phone`, `whatsapp_notification_api_key`, `enable_private_replies` to `settings`).
- **Empty Dashboard:** Seeding script `seed_db.py` created and executed to populate the database with rich Arabic mock data, displaying full metrics, graphs, analytics, conversations, and posts.
- **Audit Logs Page React Crash:** Fixed Next.js runtime crash on `/audit-logs` by conditionally formatting and stringifying the `log.details` JSON object (e.g. converting `{"ip": "192.168.1.50"}` to `"ip: 192.168.1.50"`) before rendering as a JSX child.
- **ESLint Code Quality Warnings:** Fixed TypeScript catch-block unused variables inside `/components/layout/sidebar.tsx` to align ESLint check to a fully green success state.

