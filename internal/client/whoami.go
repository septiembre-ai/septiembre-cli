package client

import "context"

// AuthUser holds the identity of the currently authenticated caller as returned
// by GET /api/v1/auth/me. Fields mirror the AuthUser set by the middleware.
type AuthUser struct {
	Sub    string   `json:"sub"`
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

// Whoami returns the identity of the caller authenticated by the current token.
func (c *Client) Whoami(ctx context.Context) (*AuthUser, error) {
	var resp AuthUser
	if err := c.get(ctx, "/api/v1/auth/me", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
