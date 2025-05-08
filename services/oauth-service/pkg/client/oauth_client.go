package client

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
)

// OAuthClient represents an OAuth2 client for communicating with the OAuth service
type OAuthClient struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	AccessToken  string
	TokenExpiry  time.Time
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

// NewOAuthClient creates a new OAuth client instance
func NewOAuthClient(baseURL, clientID, clientSecret string) *OAuthClient {
	return &OAuthClient{
		BaseURL:      baseURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// GetClientCredentialsToken obtains an access token using the client credentials grant
func (c *OAuthClient) GetClientCredentialsToken(scope string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	if scope != "" {
		data.Set("scope", scope)
	}

	return c.requestToken(data)
}

// GetPasswordToken obtains an access token using the password grant
func (c *OAuthClient) GetPasswordToken(username, password, scope string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)
	if scope != "" {
		data.Set("scope", scope)
	}

	return c.requestToken(data)
}

// RefreshToken exchanges a refresh token for a new access token
func (c *OAuthClient) RefreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	return c.requestToken(data)
}

// ValidateToken performs token introspection to check if a token is valid
func (c *OAuthClient) ValidateToken(token string) (*TokenValidationResponse, error) {
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
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("error validating token: %d %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("error validating token: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	var validationResp TokenValidationResponse
	if err := json.Unmarshal(body, &validationResp); err != nil {
		return nil, err
	}

	return &validationResp, nil
}

// RevokeToken revokes an access or refresh token
func (c *OAuthClient) RevokeToken(token, tokenTypeHint string) error {
	data := url.Values{}
	data.Set("token", token)
	if tokenTypeHint != "" {
		data.Set("token_type_hint", tokenTypeHint)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/revoke", c.BaseURL), strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return fmt.Errorf("error revoking token: %d %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("error revoking token: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	return nil
}

// CallAPI makes an authenticated API call to a protected resource
func (c *OAuthClient) CallAPI(method, path string, body io.Reader) ([]byte, error) {
	// Ensure we have a valid token
	if c.AccessToken == "" || time.Now().After(c.TokenExpiry) {
		tokenResp, err := c.GetClientCredentialsToken("")
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}
		c.AccessToken = tokenResp.AccessToken
		c.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// Make the API call
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, path), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed: %d %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Helper function to make token requests
func (c *OAuthClient) requestToken(data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/token", c.BaseURL), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+c.getBasicAuth())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return nil, fmt.Errorf("error requesting token: %d %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("error requesting token: %s - %s", errorResp.Error, errorResp.ErrorDescription)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// Helper function to create basic auth credentials
func (c *OAuthClient) getBasicAuth() string {
	auth := c.ClientID + ":" + c.ClientSecret
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
