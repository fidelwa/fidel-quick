package webhook

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Verify handles the WhatsApp webhook verification (GET /webhook).
// Meta sends a GET request with hub.mode, hub.verify_token, and hub.challenge.
func Verify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	verifyToken := os.Getenv("WHATSAPP_VERIFY_TOKEN")

	if mode == "subscribe" && token == verifyToken {
		log.Println("Webhook verified successfully")
		c.String(http.StatusOK, challenge)
		return
	}

	log.Println("Webhook verification failed")
	c.String(http.StatusForbidden, "Forbidden")
}

// Receive handles incoming WhatsApp messages (POST /webhook).
func Receive(c *gin.Context) {
	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("Error parsing webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				log.Printf("Message from %s: %s", msg.From, msg.Text.Body)
				fmt.Printf("[WhatsApp] %s: %s\n", msg.From, msg.Text.Body)
			}
		}
	}

	// Always respond 200 to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}
