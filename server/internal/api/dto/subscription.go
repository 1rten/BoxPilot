package dto

type Subscription struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	URL                string  `json:"url"`
	Type               string  `json:"type"`
	Enabled            bool    `json:"enabled"`
	AutoUpdateEnabled  bool    `json:"auto_update_enabled"`
	RefreshIntervalSec int     `json:"refresh_interval_sec"`
	Etag               string  `json:"etag,omitempty"`
	LastModified       string  `json:"last_modified,omitempty"`
	LastFetchAt        *string `json:"last_fetch_at,omitempty"`
	LastSuccessAt      *string `json:"last_success_at,omitempty"`
	LastError          *string `json:"last_error,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type CreateSubscriptionRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	Type               string `json:"type"`
	AutoUpdateEnabled  *bool  `json:"auto_update_enabled"`
	RefreshIntervalSec int    `json:"refresh_interval_sec"`
}

type UpdateSubscriptionRequest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	URL                string `json:"url"`
	Enabled            *bool  `json:"enabled"`
	AutoUpdateEnabled  *bool  `json:"auto_update_enabled"`
	RefreshIntervalSec *int   `json:"refresh_interval_sec"`
}
