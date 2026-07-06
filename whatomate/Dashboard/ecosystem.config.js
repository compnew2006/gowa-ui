module.exports = {
  apps: [{
    name: 'ai-dashboard',
    script: 'npm start',
    cwd: '/opt/ai-dashboard/dashboard',
    env: {
      NODE_ENV: 'production',
      PORT: 3000,
      DASHBOARD_ADMIN_EMAIL: 'admin@example.com',
      DASHBOARD_ADMIN_PASSWORD: 'AdminSecure2026!',
      DASHBOARD_ADMIN_NAME: 'Dashboard Admin',
      DASHBOARD_SESSION_SECRET: 'dashboard_session_secret_2026_ai_automation',
      API_URL: 'http://127.0.0.1:8000'
    }
  }]
};