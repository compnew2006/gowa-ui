const wppconnect = require('@wppconnect-team/wppconnect');
const fs = require('fs');
const path = require('path');

class SessionManager {
  constructor() {
    this.sessions = new Map();
    this.tokensDir = process.env.WPP_TOKENS_DIR || path.join(__dirname, 'tokens');
    
    // Create tokens directory if it doesn't exist
    if (!fs.existsSync(this.tokensDir)) {
      fs.mkdirSync(this.tokensDir, { recursive: true, mode: 0o700 });
    }
  }

  async createSession(sessionName) {
    if (this.sessions.has(sessionName)) {
      return { success: false, message: 'Session already exists' };
    }

    try {
      const client = await wppconnect.create({
        session: sessionName,
        tokenStore: 'file',
        folderNameToken: this.tokensDir,
        headless: 'new',
        devtools: false,
        useChrome: true,
        debug: false,
        logQR: true,
        browserArgs: ['--no-sandbox', '--disable-setuid-sandbox'],
        disableWelcome: true,
        updatesLog: false,
        autoClose: 60000,
        catchQR: (base64Qr, asciiQR, attempts, urlCode) => {
          if (process.env.WPP_LOG_QR === '1') {
            console.log(`QR Code for session ${sessionName}:`);
            console.log(asciiQR);
          }
          
          // Store QR code for API response
          if (this.sessions.has(sessionName)) {
            this.sessions.get(sessionName).qrCode = base64Qr;
            this.sessions.get(sessionName).urlCode = urlCode;
          }
        },
        statusFind: (statusSession, session) => {
          console.log(`Status for ${session}: ${statusSession}`);
          
          if (this.sessions.has(sessionName)) {
            this.sessions.get(sessionName).status = statusSession;
          }
        }
      });

      this.sessions.set(sessionName, {
        client: client,
        status: 'connecting',
        qrCode: null,
        urlCode: null,
        createdAt: new Date()
      });

      // Wait for connection
      await new Promise((resolve) => {
        const checkConnection = setInterval(() => {
          const session = this.sessions.get(sessionName);
          if (session && session.status === 'qrReadSuccess') {
            clearInterval(checkConnection);
            resolve();
          }
        }, 1000);

        // Timeout after 60 seconds
        setTimeout(() => {
          clearInterval(checkConnection);
          resolve();
        }, 60000);
      });

      return { 
        success: true, 
        message: 'Session created successfully',
        sessionName: sessionName
      };
    } catch (error) {
      console.error(`Error creating session ${sessionName}:`, error);
      this.sessions.delete(sessionName);
      return { success: false, message: error.message };
    }
  }

  getSession(sessionName) {
    return this.sessions.get(sessionName);
  }

  getAllSessions() {
    const sessions = [];
    this.sessions.forEach((session, name) => {
      sessions.push({
        name: name,
        status: session.status,
        createdAt: session.createdAt
      });
    });
    return sessions;
  }

  async closeSession(sessionName) {
    const session = this.sessions.get(sessionName);
    if (!session) {
      return { success: false, message: 'Session not found' };
    }

    try {
      await session.client.close();
      this.sessions.delete(sessionName);
      return { success: true, message: 'Session closed successfully' };
    } catch (error) {
      return { success: false, message: error.message };
    }
  }

  async sendMessage(sessionName, phone, message) {
    const session = this.sessions.get(sessionName);
    
    if (!session) {
      return { success: false, message: 'Session not found' };
    }

    if (session.status !== 'qrReadSuccess' && session.status !== 'isLogged') {
      return { success: false, message: 'Session is not connected' };
    }

    try {
      // Format phone number (add @c.us if not present)
      const cleanedPhone = String(phone).replace(/[\s+\-]/g, '');
      const formattedPhone = cleanedPhone.includes('@c.us') ? cleanedPhone : `${cleanedPhone}@c.us`;
      
      await session.client.sendText(formattedPhone, String(message).trim());
      
      return { 
        success: true, 
        message: 'Message sent successfully',
        to: phone
      };
    } catch (error) {
      console.error(`Error sending message from ${sessionName}:`, error);
      return { success: false, message: error.message };
    }
  }

  getQRCode(sessionName) {
    const session = this.sessions.get(sessionName);
    if (!session) {
      return { success: false, message: 'Session not found' };
    }

    return {
      success: true,
      qrCode: session.qrCode,
      urlCode: session.urlCode,
      status: session.status
    };
  }
}

module.exports = new SessionManager();
