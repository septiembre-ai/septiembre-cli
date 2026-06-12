package client

import (
	"context"
	"time"
)

// Org represents a Septiembre organization the authenticated user belongs to.
type Org struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// ListOrgs returns all organizations the authenticated user belongs to.
func (c *Client) ListOrgs(ctx context.Context) ([]Org, error) {
	var resp []Org
	if err := c.get(ctx, "/api/v1/orgs", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
