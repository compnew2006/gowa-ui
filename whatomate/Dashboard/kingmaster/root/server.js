const express = require('express');
const cors = require('cors');
const bodyParser = require('body-parser');
const sessionManager = require('./sessionManager');

const app = express();
const PORT = process.env.PORT || 3000;
const API_KEY = process.env.WPP_API_KEY || '';
const allowedOrigins = (process.env.WPP_ALLOWED_ORIGINS || '')
  .split(',')
  .map((origin) => origin.trim())
  .filter(Boolean);

function requireApiKey(req, res, next) {
  if (!API_KEY) {
    return next();
  }

  const provided = req.get('x-api-key') || '';
  if (provided !== API_KEY) {
    return res.status(401).json({ success: false, message: 'Unauthorized' });
  }
  return next();
}

function isValidSessionName(value) {
  return typeof value === 'string' && /^[A-Za-z0-9_-]{1,64}$/.test(value);
}

function isValidPhone(value) {
  return typeof value === 'string' && /^[0-9+@.\-_\s]{5,40}$/.test(value);
}

// Middleware
app.disable('x-powered-by');
app.use(cors({
  origin(origin, callback) {
    if (!origin || allowedOrigins.length === 0 || allowedOrigins.includes(origin)) {
      return callback(null, true);
    }
    return callback(new Error('Not allowed by CORS'));
  }
}));
app.use(bodyParser.json({ limit: '64kb' }));
app.use(bodyParser.urlencoded({ extended: true, limit: '64kb' }));
app.use(requireApiKey);

// Home route
app.get('/', (req, res) => {
  res.json({
    message: 'WPPConnect API Server',
    version: '1.0.0',
    endpoints: {
      'POST /session/start': 'Start a new WhatsApp session',
      'GET /session/:name/qr': 'Get QR code for session',
      'GET /session/:name/status': 'Get session status',
      'GET /sessions': 'Get all sessions',
      'DELETE /session/:name': 'Close a session',
      'POST /message/send': 'Send a message'
    }
  });
});

// Start a new session
app.post('/session/start', async (req, res) => {
  const { sessionName } = req.body;

  if (!isValidSessionName(sessionName)) {
    return res.status(400).json({
      success: false,
      message: 'Valid session name is required'
    });
  }

  try {
    const result = await sessionManager.createSession(sessionName);
    res.json(result);
  } catch (error) {
    console.error('Failed to start session:', error);
    res.status(500).json({
      success: false,
      message: 'Failed to start session'
    });
  }
});

// Get QR code for session
app.get('/session/:name/qr', (req, res) => {
  const { name } = req.params;
  if (!isValidSessionName(name)) {
    return res.status(400).json({ success: false, message: 'Invalid session name' });
  }
  const result = sessionManager.getQRCode(name);

  if (!result.success) {
    return res.status(404).json(result);
  }

  res.json(result);
});

// Get session status
app.get('/session/:name/status', (req, res) => {
  const { name } = req.params;
  if (!isValidSessionName(name)) {
    return res.status(400).json({ success: false, message: 'Invalid session name' });
  }
  const session = sessionManager.getSession(name);

  if (!session) {
    return res.status(404).json({
      success: false,
      message: 'Session not found'
    });
  }

  res.json({
    success: true,
    sessionName: name,
    status: session.status,
    createdAt: session.createdAt
  });
});

// Get all sessions
app.get('/sessions', (req, res) => {
  const sessions = sessionManager.getAllSessions();
  res.json({
    success: true,
    sessions: sessions
  });
});

// Close session
app.delete('/session/:name', async (req, res) => {
  const { name } = req.params;
  if (!isValidSessionName(name)) {
    return res.status(400).json({ success: false, message: 'Invalid session name' });
  }
  const result = await sessionManager.closeSession(name);

  if (!result.success) {
    return res.status(404).json(result);
  }

  res.json(result);
});

// Send message
app.post('/message/send', async (req, res) => {
  const { sessionName, phone, message } = req.body;

  if (!isValidSessionName(sessionName) || !isValidPhone(phone) || typeof message !== 'string' || message.trim().length === 0 || message.length > 4096) {
    return res.status(400).json({
      success: false,
      message: 'Valid sessionName, phone, and message are required'
    });
  }

  const result = await sessionManager.sendMessage(sessionName, phone, message);

  if (!result.success) {
    return res.status(400).json(result);
  }

  res.json(result);
});

// Start server
app.listen(PORT, '127.0.0.1', () => {
  console.log(`🚀 WPPConnect API Server running on port ${PORT}`);
  console.log(`📱 Ready to handle WhatsApp sessions`);
});
