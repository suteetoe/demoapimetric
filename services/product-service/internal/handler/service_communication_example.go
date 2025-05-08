package handler

import (
	"encoding/json"
	"net/http"
	"product-service/pkg/logger"
	"product-service/pkg/oauth"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// OAuthClient is a global variable to store the OAuth client instance
var OAuthClient *oauth.Client

// InitOAuthClient initializes the global OAuth client
func InitOAuthClient(client *oauth.Client) {
	OAuthClient = client
}

// ExampleSupplierData represents data from the supplier service
type ExampleSupplierData struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone"`
}

// GetSuppliersExample demonstrates calling another service using OAuth
func GetSuppliersExample(c echo.Context) error {
	log := logger.FromContext(c)

	// Check if OAuth client is available
	if OAuthClient == nil {
		log.Error("OAuth client not initialized")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "OAuth client not configured",
		})
	}

	log.Info("Fetching suppliers from supplier service using OAuth")

	// Make an authenticated call to the supplier service
	response, err := OAuthClient.CallAPI("GET", "http://localhost:8083/api/suppliers?limit=5", nil)
	if err != nil {
		log.Error("Failed to call supplier service", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "Failed to call supplier service",
			"details": err.Error(),
		})
	}

	// Parse the response
	var suppliers []ExampleSupplierData
	if err := json.Unmarshal(response, &suppliers); err != nil {
		log.Error("Failed to parse supplier response", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to parse supplier response",
		})
	}

	log.Info("Successfully fetched suppliers", zap.Int("count", len(suppliers)))

	// Return the response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Successfully fetched suppliers using OAuth",
		"suppliers": suppliers,
	})
}
