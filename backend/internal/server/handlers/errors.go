package handlers

import (
	"errors"
	"net/http"

	"app-booking/internal/modules/installation"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// respondErr maps domain errors to HTTP status codes. Shared by every handler
// in this package — add new domains' sentinel errors here as features grow.
func respondErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, installation.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// respondBindErr handles request-body binding failures. Field validation
// failures become a 422 with a {"errors": {field: message}} map; a malformed
// or unparseable body (which can't be attributed to a field) is a 400.
func respondBindErr(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[fe.Field()] = validationMessage(fe)
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": fields})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// validationMessage turns a single field error into a human-readable message.
func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	default:
		return fe.Field() + " is invalid"
	}
}
