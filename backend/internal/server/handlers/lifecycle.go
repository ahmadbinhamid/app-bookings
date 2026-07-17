package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"app-booking/internal/modules/installation"

	"github.com/gin-gonic/gin"
)

// SignatureMiddleware verifies the X-Flowpos-Signature header FlowPOS attaches
// to install/uninstall/webhook calls: hex(HMAC-SHA256(secret, rawBody)), where
// secret is this app's signing secret as configured in its marketplace listing.
func SignatureMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "signing secret not configured"})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "could not read body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		got := c.GetHeader("X-Flowpos-Signature")
		if got == "" || !hmac.Equal([]byte(expected), []byte(got)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
		c.Next()
	}
}

// LifecycleHandler serves the marketplace lifecycle endpoints FlowPOS calls
// directly (no tenant JWT): app install/uninstall and store event webhooks.
type LifecycleHandler struct {
	installations *installation.Service
}

func NewLifecycleHandler(installations *installation.Service) *LifecycleHandler {
	return &LifecycleHandler{installations: installations}
}

// lifecyclePayload covers the body shape FlowPOS sends to all three
// endpoints: {"event": "install"|"uninstall"|"<webhook event>", "tenant_id":
// N, "app": "app-booking", ...event-specific fields}.
type lifecyclePayload struct {
	Event    string          `json:"event"`
	TenantID uint64          `json:"tenant_id" binding:"required"`
	App      string          `json:"app"`
	APIKey   string          `json:"api_key"`
	Data     json.RawMessage `json:"data"`
}

// Install provisions (or re-activates) the tenant's installation when a
// merchant installs this app from the FlowPOS marketplace.
func (h *LifecycleHandler) Install(c *gin.Context) {
	var in lifecyclePayload
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	inst, err := h.installations.Install(in.TenantID, in.APIKey)
	if err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"installation": inst})
}

// Uninstall soft-disables the tenant's installation when a merchant removes
// this app. No data is deleted.
func (h *LifecycleHandler) Uninstall(c *gin.Context) {
	var in lifecyclePayload
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	if err := h.installations.Uninstall(in.TenantID); err != nil {
		respondErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Webhook receives store event notifications. None currently require action
// here, but the cases are stubbed out so future features have an obvious
// place to land.
func (h *LifecycleHandler) Webhook(c *gin.Context) {
	var in lifecyclePayload
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	switch in.Event {
	case "order.placed", "order.status_changed", "product.created", "product.updated", "payment.paid":
		// no-op for now
	default:
		// unknown/unhandled — still acknowledge so FlowPOS doesn't retry.
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
