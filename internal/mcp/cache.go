package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cableknitai/cableknit-cli/internal/api"
	"github.com/cableknitai/cableknit-cli/internal/config"
)

const cacheTTL = 1 * time.Hour

type Cache struct {
	manifest *api.Manifest
	loadedAt time.Time
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Load() {
	// Try API first
	client := api.NewAPIClient()
	m, err := api.FetchManifest(client)
	if err == nil {
		c.manifest = m
		c.loadedAt = time.Now()
		c.writeDisk(m)
		return
	}

	// Fallback to disk cache
	if dm := c.readDisk(); dm != nil {
		c.manifest = dm
		c.loadedAt = time.Now()
		fmt.Fprintf(os.Stderr, "mcp: using cached manifest (API unavailable)\n")
		return
	}

	// Empty manifest — tools will return "no data available"
	c.manifest = &api.Manifest{Categories: make(map[string][]api.ManifestItem)}
	fmt.Fprintf(os.Stderr, "mcp: no manifest available (API unreachable, no cache)\n")
}

func (c *Cache) Get() *api.Manifest {
	return c.manifest
}

func (c *Cache) MCPContent(key string) string {
	if c.manifest == nil {
		return ""
	}
	return c.manifest.TextContent("mcp", key)
}

func (c *Cache) MCPContentJSON(key string) json.RawMessage {
	if c.manifest == nil {
		return nil
	}
	return c.manifest.JSONContent("mcp", key)
}

func (c *Cache) ScaffoldContent(key string) string {
	if c.manifest == nil {
		return ""
	}
	return c.manifest.TextContent("scaffold", key)
}

func (c *Cache) ScaffoldContentJSON(key string) json.RawMessage {
	if c.manifest == nil {
		return nil
	}
	return c.manifest.JSONContent("scaffold", key)
}

func (c *Cache) DocContent(key string) string {
	if c.manifest == nil {
		return ""
	}
	return c.manifest.Doc(key)
}

// Disk cache

func cacheDir() string {
	return filepath.Join(config.Dir(), "mcp-cache")
}

func cacheFile() string {
	return filepath.Join(cacheDir(), "manifest.json")
}

func (c *Cache) writeDisk(m *api.Manifest) {
	dir := cacheDir()
	os.MkdirAll(dir, 0o700)

	data, err := json.Marshal(m)
	if err != nil {
		return
	}

	wrapper := struct {
		CachedAt time.Time       `json:"cached_at"`
		Data     json.RawMessage `json:"data"`
	}{
		CachedAt: time.Now(),
		Data:     data,
	}

	out, err := json.Marshal(wrapper)
	if err != nil {
		return
	}

	os.WriteFile(cacheFile(), out, 0o600)
}

func (c *Cache) readDisk() *api.Manifest {
	data, err := os.ReadFile(cacheFile())
	if err != nil {
		return nil
	}

	var wrapper struct {
		CachedAt time.Time       `json:"cached_at"`
		Data     json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil
	}

	if time.Since(wrapper.CachedAt) > 24*time.Hour {
		return nil // Stale cache — only use within 24h
	}

	var m api.Manifest
	if err := json.Unmarshal(wrapper.Data, &m); err != nil {
		return nil
	}

	return &m
}
