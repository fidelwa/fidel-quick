# fidel-quick

WhatsApp Business API webhook receiver built with Go and Gin.

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [ngrok](https://ngrok.com/download) account and CLI installed
- A [Meta Developer](https://developers.facebook.com/) app with WhatsApp product enabled

## Setup

### 1. Install dependencies

```bash
go mod download
```

### 2. Configure environment variables

```bash
cp .env.example .env
```

Edit `.env` with your values:

| Variable | Description |
|---|---|
| `WHATSAPP_VERIFY_TOKEN` | A secret string you choose. Must match what you enter in Meta's webhook config. |
| `WHATSAPP_API_TOKEN` | Your WhatsApp API access token from Meta Developer dashboard. |
| `WHATSAPP_PHONE_NUMBER_ID` | The Phone Number ID from your WhatsApp Business account. |
| `PORT` | Server port (default: `8080`). |

### 3. Start the server

```bash
go run main.go
```

### 4. Start ngrok

In a separate terminal:

```bash
ngrok http 8080
```

ngrok will display a public URL like `https://xxxx-xxxx.ngrok-free.app`. Copy this URL.

> **Note:** The ngrok URL changes every time you restart it (unless you have a paid plan with a fixed domain).

### 5. Configure the webhook in Meta

1. Go to [Meta for Developers](https://developers.facebook.com/) and open your app.
2. Navigate to **WhatsApp > Configuration**.
3. Under **Webhook**, click **Edit**:
   - **Callback URL:** `https://YOUR-NGROK-URL/webhook`
   - **Verify Token:** the same value you set in `WHATSAPP_VERIFY_TOKEN`
4. Click **Verify and Save**.
5. Under **Webhook fields**, find **`messages`** and toggle **Subscribe** on.

### 6. Test

Send a WhatsApp message to your Business number. You should see the log in your server terminal:

```
[WhatsApp] 5215512345678: Hello!
```

## Project Structure

```
fidel-quick/
  main.go              # Gin server and route setup
  webhook/
    handler.go         # GET /webhook (verification) + POST /webhook (message handling)
    types.go           # WhatsApp webhook payload structs
  .env.example         # Environment variable template
```

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/webhook` | WhatsApp webhook verification (called by Meta during setup) |
| `POST` | `/webhook` | Receives incoming WhatsApp messages |
