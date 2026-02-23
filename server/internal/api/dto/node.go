package dto

type Node struct {
	ID        string `json:"id"`
	SubID     string `json:"sub_id"`
	Tag       string `json:"tag"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type UpdateNodeRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}
