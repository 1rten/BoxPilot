package dto

type Subscription struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	Type               string   `json:"type"`
	Enabled            bool     `json:"enabled"`
	AutoUpdateEnabled  bool     `json:"auto_update_enabled"`
	RefreshIntervalSec int      `json:"refresh_interval_sec"`
	Etag               string   `json:"etag,omitempty"`
	LastModified       string   `json:"last_modified,omitempty"`
	LastFetchAt        *string  `json:"last_fetch_at,omitempty"`
	LastSuccessAt      *string  `json:"last_success_at,omitempty"`
	LastError          *string  `json:"last_error,omitempty"`
	UsedBytes          *int64   `json:"used_bytes,omitempty"`
	TotalBytes         *int64   `json:"total_bytes,omitempty"`
	RemainingBytes     *int64   `json:"remaining_bytes,omitempty"`
	UsagePercent       *float64 `json:"usage_percent,omitempty"`
	ExpireAt           *string  `json:"expire_at,omitempty"`
	ProfileWebPage     *string  `json:"profile_web_page,omitempty"`
	ProfileUpdateSec   *int     `json:"profile_update_interval_sec,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
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
