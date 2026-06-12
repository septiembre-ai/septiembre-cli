package client

import "context"

// AuthUser is the caller's profile as returned by GET /api/v1/auth/me.
// Fields mirror the cloud-api user object.
type AuthUser struct {
	ID         string `json:"id"`
	CognitoSub string `json:"cognito_sub"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  string `json:"created_at"`
}

// Whoami returns the identity of the caller authenticated by the current token.
func (c *Client) Whoami(ctx context.Context) (*AuthUser, error) {
	var resp AuthUser
	if err := c.get(ctx, "/api/v1/auth/me", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
