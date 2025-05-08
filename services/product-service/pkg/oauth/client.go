package oauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client represents an OAuth2 client for communicating with the OAuth service
type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	AccessToken  string
	TokenExpiry  time.Time
	Logger       *zap.Logger
}

// TokenResponse represents the response from the OAuth token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// TokenValidationResponse represents the response from the token introspection endpoint
type TokenValidationResponse struct {
	Active   bool   `json:"active"`
	ClientID string `json:"client_id"`
	UserID   uint   `json:"user_id,omitempty"`
	TenantID uint   `json:"tenant_id,omitempty"`
	Exp      int64  `json:"exp,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

// ErrorResponse represents an OAuth error response
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// NewClient creates a new OAuth client instance
func NewClient(baseURL, clientID, clientSecret string, logger *zap.Logger) *Client {
	return &Client{
		BaseURL:      baseURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
		Logger:       logger,
	}
}

// GetClientCredentialsToken obtains an access token using the client credentials grant
func (c *Client) GetClientCredentialsToken(scope string) (*TokenResponse, error) {
	c.Logger.Info("Requesting client credentials token",
		zap.String("client_id", c.ClientID),
		zap.String("scope", scope))

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	if scope != "" {
		data.Set("scope", scope)
	}

	return c.requestToken(data)
}

// ValidateToken performs token introspection to check if a token is valid
func (c *Client) ValidateToken(token string) (*TokenValidationResponse, error) {
	c.Logger.Info("Validating token",
		zap.String("client_id", c.ClientID))

	data := url.Values{}
	data.Set("token", token)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/introspect", c.BaseURL), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("Token validation request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger.Error("Failed to read token validation response", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			c.Logger.Error("Failed to parse error response",
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)))
			return nil, fmt.Errorf("error validating token: %d %s", resp.StatusCode, string(body))
		}
		c.Logger.Error("Token validation failed",
			zap.String("error", errorResp.Error),
			zap.String("description", errorResp.ErrorDescription))
		return nil, fmt.Errorf("error validating token: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	var validationResp TokenValidationResponse
	if err := json.Unmarshal(body, &validationResp); err != nil {
		c.Logger.Error("Failed to parse validation response", zap.Error(err))
		return nil, err
	}

	c.Logger.Info("Token validation result",
		zap.Bool("active", validationResp.Active),
		zap.String("client_id", validationResp.ClientID),
		zap.String("scope", validationResp.Scope))

	return &validationResp, nil
}

// CallAPI makes an authenticated API call to a protected resource
func (c *Client) CallAPI(method, path string, body io.Reader) ([]byte, error) {
	// Ensure we have a valid token
	if c.AccessToken == "" || time.Now().After(c.TokenExpiry) {
		c.Logger.Info("Access token not available or expired, requesting new token")
		tokenResp, err := c.GetClientCredentialsToken("read write")
		if err != nil {
			c.Logger.Error("Failed to get access token", zap.Error(err))
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		c.AccessToken = tokenResp.AccessToken
		c.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		c.Logger.Info("New access token acquired",
			zap.Time("expiry", c.TokenExpiry))
	}

	// Make the API call
	c.Logger.Info("Making API call",
		zap.String("method", method),
		zap.String("path", path))

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	if err != nil {
		c.Logger.Error("Failed to create request", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("API request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger.Error("Failed to read response body", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode >= 400 {
		c.Logger.Error("API request returned error status",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)))
		return nil, fmt.Errorf("API request failed: %d %s", resp.StatusCode, string(respBody))
	}

	c.Logger.Info("API call successful", zap.Int("status", resp.StatusCode))
	return respBody, nil
}

// Helper function to make token requests
func (c *Client) requestToken(data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/token", c.BaseURL), bytes.NewBufferString(data.Encode()))
	if err != nil {
		c.Logger.Error("Failed to create token request", zap.Error(err))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		c.Logger.Error("Token request failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger.Error("Failed to read token response", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			c.Logger.Error("Failed to parse error response",
				zap.Int("status_code", resp.StatusCode),
				zap.String("response", string(body)))
			return nil, fmt.Errorf("error requesting token: %d %s", resp.StatusCode, string(body))
		}
		c.Logger.Error("Token request error",
			zap.String("error", errorResp.Error),
			zap.String("description", errorResp.ErrorDescription))
		return nil, fmt.Errorf("error requesting token: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		c.Logger.Error("Failed to parse token response", zap.Error(err))
		return nil, err
	}

	c.Logger.Info("Token request successful")
	return &tokenResp, nil
}

// Helper function to create basic auth credentials
func (c *Client) getBasicAuth() string {
	auth := c.ClientID + ":" + c.ClientSecret

	//fmt.Println("Basic Auth: ", auth)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
