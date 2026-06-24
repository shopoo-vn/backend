package model

import "time"

// Media is the metadata record for an uploaded image and its derived sizes.
// Sizes maps a size name (e.g. "thumb", "medium", "full") to the object key in
// the bucket (e.g. "media/<id>/thumb.jpg"). It is persisted as JSONB.
type Media struct {
	ID           string            `json:"id"`
	OwnerID      string            `json:"owner_id"`
	OriginalName string            `json:"original_name,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
	Sizes        map[string]string `json:"-"`
	CreatedAt    time.Time         `json:"created_at"`
}
