# 🛡️ Commercialization & Security Plan for Whatomate

## 🎯 Objective

Secure the **Whatomate** application to ensure customers must maintain an active monthly subscription to use it. This prevents unauthorized usage and ensures consistent revenue.

---

## 🏗️ Architecture Overview

To achieve "Stop using if not pay", we must move away from a purely standalone disconnected app. The application must "phone home" to verify its license status.

### The Ecosystem

1.  **Whatomate Instance (User's Machine)**: The software they download.
2.  **License Server (You Host This)**: A small, separate API that tracks who has paid.
3.  **Stripe (Payment Processor)**: Handles the monthly billing and tells your License Server when a payment fails.

---

## 🔒 1. Security Implementation (The "Lock")

### A. The License Key System

Instead of a simple password, we will use **Cryptographically Signed License Keys (JWTs)**.

- **How it works**: You generate a key that contains the customer's ID and an expiration date.
- **Security**: The key is signed with your _Private Key_. The app contains only the _Public Key_. The app can verify the key is valid but cannot generate new ones.

### B. The "Call Home" Protocol (Heartbeat)

The application will not trust the local date/time (users can change their clock). It must check with your server.

**Logic Flow in `whatomate`:**

1.  **Startup**: App checks if a `LICENSE_KEY` exists in config.
2.  **Verification**:
    - Decodes the key.
    - Verifies the signature using the embedded Public Key.
    - Sends a request to `https://license.your-domain.com/v1/verify` with the key.
3.  **Server Response**:
    - `200 OK`: Active. Update local "Last Successful Check" timestamp.
    - `403 Forbidden`: Payment failed/Subscription cancelled. **Lock App**.
    - `Connection Error`: Enter **Grace Period** (e.g., 72 hours). If no internet for >72h, **Lock App**.

### C. Code Obfuscation

To prevent a developer customer from simply finding the `if (!isPaid) { exit() }` line and deleting it:

1.  **Go Obfuscation**: Use [garble](https://github.com/burrowers/garble) to build your production binaries. It renames functions and variables to random hashes (`a1`, `b2`) and strips debugging info.
    ```bash
    go install mvdan.cc/garble@latest
    garble -literals -tiny build .
    ```
2.  **Critical Checks Scattering**: Do not have a single `checkLicense()` function. Embed the check in 3-5 critical places (e.g., `main.go`, `SendMessage` handler, `Database` connection). If any fail, the app stops working.

---

## 💳 2. Monthly Payment Plan (Stripe)

We will use **Stripe Subscriptions** to automate everything.

### Workflow:

1.  **Customer Buys**: Customer goes to your checkout page (Stripe Checkout) and pays $XX/month.
2.  **Webhook Trigger**: Stripe sends a `checkout.session.completed` event to your License Server.
3.  **Key Generation**: Your License Server:
    - Creates a new user record.
    - Generates a License Key.
    - Emails the key to the customer.
4.  **Recurring Payment**:
    - **Success**: Stripe sends `invoice.paid`. Server updates `expires_at` date for that user.
    - **Failure**: Stripe sends `invoice.payment_failed`. Server marks status as `past_due`. Next time the user's app calls home, it gets a `403` and locks.

---

## 🛠️ Implementation Guide (Step-by-Step)

### Phase 1: The License Server (New Microservice)

_You need a simple backend (Node.js, Go, or Python) hosted on a cheap VPS or Cloud Function._

- **Database**: Store `license_key`, `customer_email`, `status`, `expires_at`.
- **Endpoint `POST /verify`**:
  - Input: `{ key: "..." }`
  - Logic: Look up key -> Check if `status == 'active'` AND `expires_at > now`.
  - Output: `{ valid: true }` or `{ valid: false, reason: "payment_failed" }`

### Phase 2: Stripe Integration

- Create a "Product" in Stripe Dashboard (e.g., "Whatomate Pro - Monthly").
- Set up a Webhook output to your License Server URL.
- Handle events: `checkout.session.completed`, `customer.subscription.deleted`, `invoice.payment_succeeded`.

### Phase 3: Modify Whatomate Core

Add the enforcement logic to `cmd/whatomate/main.go`.

```go
// PSEUDO-CODE EXAMPLE
func verifyLicense() {
    key := config.Get("LICENSE_KEY")

    // 1. Local Crypto Check
    if !validateSignature(key) {
        log.Fatal("❌ Invalid License Key Signature")
    }

    // 2. Remote Heartbeat
    status, err := http.Post("https://api.yourbrand.com/verify", key)

    if err != nil {
        // Internet down? Check Grace Period
        if time.Since(lastSuccess) < 72*time.Hour {
            return // Allow temporary offline
        }
        log.Fatal("❌ Cannot verify license. Connect to internet.")
    }

    if status == "expired" {
        log.Fatal("❌ Subscription expired. Please renew at yourbrand.com")
    }

    saveLastSuccessTime(time.Now())
}
```

### Phase 4: Distribution

- Distribute **Compiled Binaries** only. Never the source code.
- Use Docker images where the binary is the entrypoint and source code is NOT included.

---

## 🚦 Recommended Deployment Stack

- **Obfuscation tool**: `garble`
- **License Server**: A simple Go HTTP server using `SQLite` or `PostgreSQL`.
- **Payments**: Stripe Billing (Starter tier is free/cheap).

## 📝 Next Steps

1.  **Approve this plan?** I can help you generate the boilerplate for the License Server or the specific Go code for the "Heartbeat" check in Whatomate.
