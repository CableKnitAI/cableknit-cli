package api

import (
	"encoding/json"
	"fmt"
	"sync"
)

// ManifestItem represents a single content entry from the API.
type ManifestItem struct {
	Key         string          `json:"key"`
	Content     json.RawMessage `json:"content"`
	ContentType string          `json:"content_type"`
}

// ManifestResponse is the top-level API response.
type ManifestResponse struct {
	Manifest map[string][]ManifestItem `json:"manifest"`
}

// Manifest holds fetched CLI content grouped by category.
type Manifest struct {
	Categories map[string][]ManifestItem
}

// singleton manifest with lazy fetch
var (
	globalManifest *Manifest
	manifestOnce   sync.Once
	manifestErr    error
)

// FetchManifest calls GET /api/v1/cli/manifest and returns the parsed result.
func FetchManifest(client APIClient) (*Manifest, error) {
	var resp ManifestResponse
	if err := client.JSON("GET", "/api/v1/cli/manifest", nil, &resp); err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	return &Manifest{Categories: resp.Manifest}, nil
}

// LoadManifest fetches once and caches globally.
func LoadManifest(client APIClient) (*Manifest, error) {
	manifestOnce.Do(func() {
		globalManifest, manifestErr = FetchManifest(client)
	})
	return globalManifest, manifestErr
}

// GetManifest returns the cached manifest (nil if not loaded).
func GetManifest() *Manifest {
	return globalManifest
}

// Doc returns a text content entry by key, or empty string if not found.
func (m *Manifest) Doc(key string) string {
	if m == nil {
		return ""
	}
	items, ok := m.Categories["docs"]
	if !ok {
		return ""
	}
	for _, item := range items {
		if item.Key == key {
			var s string
			if err := json.Unmarshal(item.Content, &s); err != nil {
				// Content might be a raw string without quotes
				return string(item.Content)
			}
			return s
		}
	}
	return ""
}

// JSONContent returns raw JSON content by category and key.
func (m *Manifest) JSONContent(category, key string) json.RawMessage {
	if m == nil {
		return nil
	}
	items, ok := m.Categories[category]
	if !ok {
		return nil
	}
	for _, item := range items {
		if item.Key == key {
			return item.Content
		}
	}
	return nil
}

// TextContent returns text content by category and key.
func (m *Manifest) TextContent(category, key string) string {
	if m == nil {
		return ""
	}
	items, ok := m.Categories[category]
	if !ok {
		return ""
	}
	for _, item := range items {
		if item.Key == key {
			var s string
			if err := json.Unmarshal(item.Content, &s); err != nil {
				return string(item.Content)
			}
			return s
		}
	}
	return ""
}

// HasContent checks if a category+key exists.
func (m *Manifest) HasContent(category, key string) bool {
	if m == nil {
		return false
	}
	items, ok := m.Categories[category]
	if !ok {
		return false
	}
	for _, item := range items {
		if item.Key == key {
			return true
		}
	}
	return false
}
