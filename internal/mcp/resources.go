package mcp

import "encoding/json"

func allResources() []Resource {
	return []Resource{
		{URI: "cableknit://schema/plugin.json", Name: "Plugin Manifest Schema", Description: "Full plugin.json schema with field descriptions, examples, and common mistakes.", MimeType: "application/json"},
		{URI: "cableknit://schema/automation", Name: "Automation Template Schema", Description: "Automation template schema with state/transition examples.", MimeType: "application/json"},
		{URI: "cableknit://schema/blueprint", Name: "Artifact Blueprint Schema", Description: "Artifact blueprint schema with per-type examples.", MimeType: "application/json"},
		{URI: "cableknit://schema/data-source", Name: "Data Source Schema", Description: "Data source schema with examples for all 3 source types.", MimeType: "application/json"},
		{URI: "cableknit://schema/skill", Name: "Skill Definition Schema", Description: "Skill schema with prompt safety rules.", MimeType: "application/json"},
		{URI: "cableknit://docs/cli-guide", Name: "CLI Guide", Description: "Full CableKnit CLI documentation.", MimeType: "text/plain"},
		{URI: "cableknit://docs/app-guidelines", Name: "App Review Guidelines", Description: "What we look for when reviewing plugins.", MimeType: "text/plain"},
		{URI: "cableknit://examples/invoice-sorter", Name: "Invoice Sorter Example", Description: "Complete sample plugin bundle with automation, workflow, and docs.", MimeType: "text/plain"},
	}
}

var resourceKeyMap = map[string]struct {
	category string
	key      string
}{
	"cableknit://schema/plugin.json": {"mcp", "plugin_schema_annotated"},
	"cableknit://schema/automation":  {"mcp", "automation_schema_annotated"},
	"cableknit://schema/blueprint":   {"mcp", "blueprint_schema_annotated"},
	"cableknit://schema/data-source": {"mcp", "data_source_schema_annotated"},
	"cableknit://schema/skill":       {"mcp", "skill_schema_annotated"},
	"cableknit://docs/cli-guide":     {"docs", "readme_text"},
	"cableknit://docs/app-guidelines": {"mcp", "app_guidelines"},
	"cableknit://examples/invoice-sorter": {"docs", "sample_plugin_text"},
}

func (s *Server) readResource(params ResourceReadParams) ResourceReadResult {
	mapping, ok := resourceKeyMap[params.URI]
	if !ok {
		return ResourceReadResult{
			Contents: []ResourceContent{{
				URI:  params.URI,
				Text: "Resource not found: " + params.URI,
			}},
		}
	}

	var text string
	if mapping.category == "docs" {
		text = s.cache.DocContent(mapping.key)
	} else {
		// For JSON content, pretty-print it
		raw := s.cache.Get().JSONContent(mapping.category, mapping.key)
		if raw != nil {
			pretty, err := json.MarshalIndent(json.RawMessage(raw), "", "  ")
			if err == nil {
				text = string(pretty)
			} else {
				text = string(raw)
			}
		} else {
			text = s.cache.Get().TextContent(mapping.category, mapping.key)
		}
	}

	if text == "" {
		text = "Content not available. Check API connection."
	}

	mimeType := "text/plain"
	if mapping.category == "mcp" && mapping.key != "app_guidelines" {
		mimeType = "application/json"
	}

	return ResourceReadResult{
		Contents: []ResourceContent{{
			URI:      params.URI,
			MimeType: mimeType,
			Text:     text,
		}},
	}
}
