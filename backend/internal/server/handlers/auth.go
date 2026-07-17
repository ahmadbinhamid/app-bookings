package handlers

import (
	"net/http"
	"strings"
	"time"

	"app-booking/internal/auth"

	"github.com/gin-gonic/gin"
)

const claimsKey = "fp_claims"

// AuthMiddleware validates the Bearer JWT and stashes the claims in context.
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := auth.Parse(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

func claimsFrom(c *gin.Context) *auth.Claims {
	if v, ok := c.Get(claimsKey); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims
		}
	}
	return nil
}

// tenantID returns the calling tenant's ID from the validated JWT claims (0
// if unset — should not happen once AuthMiddleware has run).
func tenantID(c *gin.Context) uint64 {
	if claims := claimsFrom(c); claims != nil {
		return claims.TenantID
	}
	return 0
}

// DevTokenHandler mints JWTs for local testing. It is only mounted when dev
// tokens are enabled (JWT_DEV_TOKENS=true) and must be disabled in prod — the
// real token is issued by the main FlowPOS system.
type DevTokenHandler struct {
	secret string
}

func NewDevTokenHandler(secret string) *DevTokenHandler {
	return &DevTokenHandler{secret: secret}
}

func (h *DevTokenHandler) Mint(c *gin.Context) {
	var in struct {
		TenantID  uint64 `json:"tenant_id" binding:"required"`
		UserID    uint64 `json:"user_id"`
		UserEmail string `json:"user_email"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respondBindErr(c, err)
		return
	}
	token, err := auth.Generate(auth.Claims{
		TenantID:  in.TenantID,
		UserID:    in.UserID,
		UserEmail: in.UserEmail,
	}, h.secret, 30*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not mint token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
