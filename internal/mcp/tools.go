package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/config"
)

func allTools() []Tool {
	return []Tool{
		// Reference / Lookup
		{
			Name:        "get_plugin_schema",
			Description: "Full plugin.json schema — required/optional fields, enums, constraints, common mistakes, and annotated example.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_automation_schema",
			Description: "Automation template schema — states, transitions, action types, condition types, and annotated example.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_blueprint_schema",
			Description: "Artifact blueprint schema — types, content_schema limits, defaults, and example.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_data_source_schema",
			Description: "Data source schema — three source types (data_store, connector, static) with examples for each.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_skill_schema",
			Description: "Skill definition schema — fields, prompt safety rules, injection restrictions, and example.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_docs_schema",
			Description: "Documentation page format — YAML frontmatter fields, markdown body rules, and example.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "list_platform_tools",
			Description: "Available platform tools with descriptions and parameter schemas. These are shared capabilities (employee lookup, decisions, spreadsheets, etc.) that plugins can request.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"enum":        []string{"platform", "contextual"},
						"description": "Filter by category. Omit for all tools.",
					},
				},
			},
		},
		{
			Name:        "list_connectors",
			Description: "Available connectors for requirements.connectors in plugin.json — name, slug, category, auth type, status.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "list_industries",
			Description: "Valid industry slugs and display names for the industry field in plugin.json.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "list_categories",
			Description: "Valid plugin and automation categories.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_bundle_limits",
			Description: "All size and count constraints for CableKnit bundles — max skills, automations, docs, images, file sizes, etc.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_app_guidelines",
			Description: "Full app review guidelines — what gets approved, what gets rejected, pricing rules, submission process.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_workflow_patterns",
			Description: "Common automation workflow patterns — intake routing, approval chains, escalation, notification fan-out, and more with examples.",
			InputSchema: emptySchema(),
		},
		{
			Name:        "get_cli_docs",
			Description: "CableKnit CLI documentation — commands, flags, workflows.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"topic": map[string]any{
						"type":        "string",
						"description": "Optional topic to filter (e.g. 'validate', 'push', 'pricing'). Omit for full docs.",
					},
				},
			},
		},
		{
			Name:        "whoami",
			Description: "Current auth status — whether the developer is logged in, publisher info, email.",
			InputSchema: emptySchema(),
		},

		// Validation / Debugging
		{
			Name:        "validate_plugin_json",
			Description: "Validate plugin.json content against the schema. Returns pass/fail with errors and fix suggestions.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"content"},
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "The plugin.json content as a JSON string.",
					},
				},
			},
		},
		{
			Name:        "validate_automation",
			Description: "Validate an automation template JSON against the schema. Returns pass/fail with errors and fix suggestions.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"content"},
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "The automation template JSON content as a string.",
					},
				},
			},
		},
		{
			Name:        "explain_error",
			Description: "Explain a CableKnit validation error in plain language and suggest how to fix it.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"error_message"},
				"properties": map[string]any{
					"error_message": map[string]any{
						"type":        "string",
						"description": "The validation error message to explain.",
					},
				},
			},
		},
		{
			Name:        "describe_automation",
			Description: "Describe an automation workflow in plain English. Takes a workflow_definition JSON (or full automation JSON) and produces a numbered, human-readable flow description showing states, transitions, conditions, and actions.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"content"},
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "JSON string of workflow_definition OR full automation JSON.",
					},
				},
			},
		},
		{
			Name:        "validate_bundle",
			Description: "Run full bundle validation on a plugin directory. Wraps `cableknit validate --json` and returns structured results with errors and warnings.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the bundle directory. Defaults to current working directory if omitted.",
					},
				},
			},
		},
		{
			Name:        "check_prompt_injection",
			Description: "Check if a system prompt would be flagged by CableKnit's prompt injection scanner.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"prompt"},
				"properties": map[string]any{
					"prompt": map[string]any{
						"type":        "string",
						"description": "The system prompt text to check.",
					},
				},
			},
		},

		// Generation
		{
			Name:        "generate_plugin_json",
			Description: "Generate a valid plugin.json from a description. Returns complete plugin.json content ready to write to disk.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "What the plugin should do.",
					},
					"tier": map[string]any{
						"type":        "string",
						"enum":        []string{"starter", "industry", "custom"},
						"description": "Plugin tier. Defaults to 'industry'.",
					},
					"industry": map[string]any{
						"type":        "string",
						"description": "Industry slug (use list_industries to see valid values).",
					},
				},
			},
		},
		{
			Name:        "generate_automation",
			Description: "Generate a valid automation template JSON from a description. Returns complete automation JSON ready to write to automations/*.json.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description", "trigger_type", "plugin_slug"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "What the automation should do.",
					},
					"trigger_type": map[string]any{
						"type":        "string",
						"enum":        []string{"inbound_email", "webhook", "schedule", "event"},
						"description": "What starts the automation.",
					},
					"plugin_slug": map[string]any{
						"type":        "string",
						"description": "The parent plugin's slug.",
					},
				},
			},
		},
		{
			Name:        "generate_blueprint",
			Description: "Generate an artifact blueprint JSON. Returns complete blueprint JSON ready to write to blueprints/*.json.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description", "artifact_type", "plugin_slug"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "What the blueprint produces.",
					},
					"artifact_type": map[string]any{
						"type":        "string",
						"enum":        []string{"document", "spreadsheet", "csv", "email_draft", "schedule", "report", "image", "json_data", "other", "chart", "options_selection"},
						"description": "Type of artifact.",
					},
					"plugin_slug": map[string]any{
						"type":        "string",
						"description": "The parent plugin's slug.",
					},
				},
			},
		},
		{
			Name:        "generate_data_source",
			Description: "Generate a data source definition. Returns complete JSON ready to write to tools/*.json.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description", "source_type", "plugin_slug"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "What the data source provides.",
					},
					"source_type": map[string]any{
						"type":        "string",
						"enum":        []string{"data_store", "connector", "static"},
						"description": "Source type.",
					},
					"plugin_slug": map[string]any{
						"type":        "string",
						"description": "The parent plugin's slug.",
					},
				},
			},
		},
		{
			Name:        "generate_skill",
			Description: "Generate a skill definition with a safe system prompt. Returns complete JSON ready to write to skills/*.json.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description", "plugin_slug"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "What the skill should help users do.",
					},
					"plugin_slug": map[string]any{
						"type":        "string",
						"description": "The parent plugin's slug.",
					},
				},
			},
		},
		{
			Name:        "generate_doc_page",
			Description: "Generate a documentation page with YAML frontmatter. Returns complete markdown ready to write to docs/*.md.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"title", "topic", "plugin_slug"},
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Page title.",
					},
					"topic": map[string]any{
						"type":        "string",
						"description": "What the page covers.",
					},
					"plugin_slug": map[string]any{
						"type":        "string",
						"description": "The parent plugin's slug.",
					},
				},
			},
		},
		{
			Name:        "scaffold_bundle",
			Description: "Generate a complete plugin bundle from a high-level idea. Returns structured JSON of all files and directory layout for the agent to write to disk.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"description", "name"},
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"description": "High-level description of what the plugin should do.",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "Plugin display name.",
					},
					"industry": map[string]any{
						"type":        "string",
						"description": "Industry slug (use list_industries to see valid values).",
					},
				},
			},
		},
	}
}

func (s *Server) callTool(params ToolCallParams) ToolResult {
	switch params.Name {

	// Reference / Lookup
	case "get_plugin_schema":
		return s.schemaResult("plugin_schema_annotated")
	case "get_automation_schema":
		return s.schemaResult("automation_schema_annotated")
	case "get_blueprint_schema":
		return s.schemaResult("blueprint_schema_annotated")
	case "get_data_source_schema":
		return s.schemaResult("data_source_schema_annotated")
	case "get_skill_schema":
		return s.schemaResult("skill_schema_annotated")
	case "get_docs_schema":
		return s.schemaResult("docs_schema_annotated")

	case "list_platform_tools":
		return s.listPlatformTools(params.Arguments)
	case "list_connectors":
		return s.listConnectors()
	case "list_industries":
		return s.listIndustries()
	case "list_categories":
		return s.listCategories()
	case "get_bundle_limits":
		return s.getBundleLimits()
	case "get_app_guidelines":
		return s.getAppGuidelines()
	case "get_workflow_patterns":
		return s.schemaResult("workflow_patterns")
	case "get_cli_docs":
		return s.getCliDocs(params.Arguments)
	case "whoami":
		return s.getWhoami()

	// Validation / Debugging
	case "validate_plugin_json":
		return s.validatePluginJSON(params.Arguments)
	case "validate_automation":
		return s.validateAutomation(params.Arguments)
	case "explain_error":
		return s.explainError(params.Arguments)
	case "describe_automation":
		return s.describeAutomation(params.Arguments)
	case "validate_bundle":
		return s.validateBundle(params.Arguments)
	case "check_prompt_injection":
		return s.checkPromptInjection(params.Arguments)

	// Generation
	case "generate_plugin_json":
		return s.generatePluginJSON(params.Arguments)
	case "generate_automation":
		return s.generateAutomation(params.Arguments)
	case "generate_blueprint":
		return s.generateBlueprint(params.Arguments)
	case "generate_data_source":
		return s.generateDataSource(params.Arguments)
	case "generate_skill":
		return s.generateSkill(params.Arguments)
	case "generate_doc_page":
		return s.generateDocPage(params.Arguments)
	case "scaffold_bundle":
		return s.scaffoldBundle(params.Arguments)

	default:
		return textError(fmt.Sprintf("Unknown tool: %s", params.Name))
	}
}

// --- Reference tool implementations ---

func (s *Server) schemaResult(key string) ToolResult {
	raw := s.cache.MCPContentJSON(key)
	if raw == nil {
		return textError("Schema not available. Try again after connecting to the API.")
	}

	pretty, err := json.MarshalIndent(json.RawMessage(raw), "", "  ")
	if err != nil {
		return textResult(string(raw))
	}
	return textResult(string(pretty))
}

func (s *Server) listPlatformTools(args map[string]any) ToolResult {
	// Try live API first
	client := api.NewAPIClient()
	var resp api.PlatformToolsResponse
	if err := client.JSON("GET", "/api/v1/cli/platform_tools", nil, &resp); err == nil {
		tools := resp.PlatformTools
		if cat, ok := args["category"].(string); ok && cat != "" {
			var filtered []api.PlatformTool
			for _, t := range tools {
				if t.Category == cat {
					filtered = append(filtered, t)
				}
			}
			tools = filtered
		}
		return jsonResult(tools)
	}

	// Fallback to cached demo data
	raw := s.cache.Get().JSONContent("demo_data", "mock_platform_tools")
	if raw != nil {
		return textResult(string(raw))
	}
	return textError("Platform tools not available. Check API connection.")
}

func (s *Server) listConnectors() ToolResult {
	client := api.NewAPIClient()
	var resp api.ConnectorsResponse
	if err := client.JSON("GET", "/api/v1/cli/connectors", nil, &resp); err == nil {
		return jsonResult(resp.Connectors)
	}

	raw := s.cache.Get().JSONContent("demo_data", "mock_connectors")
	if raw != nil {
		return textResult(string(raw))
	}
	return textError("Connectors not available. Check API connection.")
}

func (s *Server) listIndustries() ToolResult {
	raw := s.cache.ScaffoldContentJSON("industries")
	if raw != nil {
		return textResult(string(raw))
	}
	return textError("Industries not available.")
}

func (s *Server) listCategories() ToolResult {
	raw := s.cache.ScaffoldContentJSON("categories")
	if raw == nil {
		return textError("Categories not available.")
	}

	result := map[string]json.RawMessage{
		"plugin_categories":     raw,
		"automation_categories": mustJSON([]string{"intake", "notification", "escalation", "communication", "reporting"}),
	}
	return jsonResult(result)
}

func (s *Server) getBundleLimits() ToolResult {
	limits := map[string]any{
		"max_bundle_size":           "10MB",
		"max_skills":               20,
		"max_automations":          10,
		"max_docs":                 50,
		"max_blueprints":           20,
		"max_tools":                10,
		"max_images":               5,
		"max_image_size":           "2MB",
		"max_blueprint_schema":     "4KB",
		"max_static_data":          "64KB",
		"allowed_image_types":      []string{"png", "jpg", "jpeg", "webp"},
		"allowed_dirs":             []string{"plugin.json", "README.md", "skills/*.json", "automations/*.json", "blueprints/*.json", "tools/*.json", "docs/*.md", "images/*.(png|jpg|jpeg|webp)"},
		"icon_required":            true,
		"icon_format":              "512x512 PNG at images/icon.png",
		"settings_types":           []string{"string", "email", "number", "boolean", "url", "file"},
		"slug_format":              "lowercase alphanumeric and hyphens only (a-z, 0-9, -)",
		"version_format":           "semver (MAJOR.MINOR.PATCH, e.g. 1.0.0)",
		"pricing_model":            "monthly only",
		"cost_floor_formula":       "ai_actions × 2000 tokens × $0.000001/token × 3× margin × 500 runs/mo",
		"max_data_sources_per_plugin": 10,
	}
	return jsonResult(limits)
}

func (s *Server) getAppGuidelines() ToolResult {
	text := s.cache.MCPContent("app_guidelines")
	if text != "" {
		return textResult(text)
	}
	return textError("App guidelines not available.")
}

func (s *Server) getCliDocs(args map[string]any) ToolResult {
	docs := s.cache.DocContent("readme_text")
	if docs == "" {
		return textError("CLI docs not available.")
	}

	topic, _ := args["topic"].(string)
	if topic == "" {
		return textResult(docs)
	}

	// Simple section search
	topic = strings.ToLower(topic)
	lines := strings.Split(docs, "\n")
	var result []string
	capturing := false
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, topic) {
			capturing = true
		} else if capturing && len(strings.TrimSpace(line)) > 0 && !strings.HasPrefix(strings.TrimSpace(line), " ") && !strings.HasPrefix(strings.TrimSpace(line), "-") {
			// New top-level section, stop capturing
			if len(result) > 5 {
				break
			}
		}
		if capturing {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return textResult(fmt.Sprintf("No section found for topic '%s'. Here are the full docs:\n\n%s", topic, docs))
	}
	return textResult(strings.Join(result, "\n"))
}

func (s *Server) getWhoami() ToolResult {
	token := config.Token()
	if token == "" {
		result := map[string]any{
			"authenticated": false,
			"message":       "Not logged in. Run `cableknit login` to authenticate.",
			"note":          "All MCP tools work without authentication. To publish plugins, apply for a developer account at cableknit.ai/developer/apply.",
		}
		return jsonResult(result)
	}

	client := api.NewAPIClient()
	var user api.User
	if err := client.JSON("GET", "/api/v1/cli/me", nil, &user); err != nil {
		return jsonResult(map[string]any{
			"authenticated": false,
			"message":       "Token present but API unreachable.",
		})
	}

	return jsonResult(map[string]any{
		"authenticated":    true,
		"email":            user.Email,
		"name":             user.Name,
		"publisher_status": user.PublisherStatus,
	})
}

// --- Validation tool implementations ---

func (s *Server) validatePluginJSON(args map[string]any) ToolResult {
	content, _ := args["content"].(string)
	if content == "" {
		return textError("content is required")
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return jsonResult(map[string]any{
			"valid":  false,
			"errors": []map[string]string{{"field": "json", "message": fmt.Sprintf("Invalid JSON: %s", err.Error())}},
		})
	}

	var errors []map[string]string

	// Required fields
	for _, field := range []string{"name", "slug", "version", "description"} {
		if data[field] == nil || data[field] == "" {
			errors = append(errors, map[string]string{"field": field, "message": "required", "fix": fmt.Sprintf("Add a '%s' field", field)})
		}
	}

	// Slug format
	if slug, ok := data["slug"].(string); ok && slug != "" {
		for _, c := range slug {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				errors = append(errors, map[string]string{"field": "slug", "message": "invalid format", "fix": "Use only lowercase a-z, 0-9, and hyphens"})
				break
			}
		}
	}

	// Version format
	if v, ok := data["version"].(string); ok && v != "" {
		parts := strings.Split(v, ".")
		if len(parts) != 3 {
			errors = append(errors, map[string]string{"field": "version", "message": "must be semver (e.g. 1.0.0)", "fix": "Use MAJOR.MINOR.PATCH format"})
		}
	}

	// Author
	author, _ := data["author"].(map[string]any)
	if author == nil || author["name"] == nil || author["name"] == "" {
		errors = append(errors, map[string]string{"field": "author.name", "message": "required", "fix": "Add author: { name: \"Your Name\" }"})
	}

	// Tier
	tier, _ := data["tier"].(string)
	if tier != "" && tier != "starter" && tier != "industry" && tier != "custom" {
		errors = append(errors, map[string]string{"field": "tier", "message": "must be starter, industry, or custom"})
	}

	// Visibility
	vis, _ := data["visibility"].(string)
	if vis != "" && vis != "private" && vis != "unlisted" && vis != "public" {
		errors = append(errors, map[string]string{"field": "visibility", "message": "must be private, unlisted, or public"})
	}

	// Pricing
	pricing, _ := data["pricing"].(map[string]any)
	if pricing != nil {
		if model, _ := pricing["model"].(string); model != "monthly" {
			errors = append(errors, map[string]string{"field": "pricing.model", "message": "must be 'monthly'"})
		}
		switch pc := pricing["price_cents"].(type) {
		case float64:
			if pc < 0 {
				errors = append(errors, map[string]string{"field": "pricing.price_cents", "message": "must be >= 0"})
			}
		default:
			errors = append(errors, map[string]string{"field": "pricing.price_cents", "message": "must be an integer >= 0"})
		}
	}

	valid := len(errors) == 0
	return jsonResult(map[string]any{"valid": valid, "errors": errors})
}

func (s *Server) validateAutomation(args map[string]any) ToolResult {
	content, _ := args["content"].(string)
	if content == "" {
		return textError("content is required")
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return jsonResult(map[string]any{
			"valid":  false,
			"errors": []map[string]string{{"field": "json", "message": fmt.Sprintf("Invalid JSON: %s", err.Error())}},
		})
	}

	var errors []map[string]string

	for _, field := range []string{"name", "slug", "category", "trigger_type"} {
		if data[field] == nil || data[field] == "" {
			errors = append(errors, map[string]string{"field": field, "message": "required"})
		}
	}

	if cat, ok := data["category"].(string); ok {
		validCats := map[string]bool{"intake": true, "notification": true, "escalation": true, "communication": true, "reporting": true}
		if !validCats[cat] {
			errors = append(errors, map[string]string{"field": "category", "message": "must be one of: intake, notification, escalation, communication, reporting"})
		}
	}

	if tt, ok := data["trigger_type"].(string); ok {
		validTT := map[string]bool{"inbound_email": true, "webhook": true, "schedule": true, "event": true}
		if !validTT[tt] {
			errors = append(errors, map[string]string{"field": "trigger_type", "message": "must be one of: inbound_email, webhook, schedule, event"})
		}
	}

	if data["workflow_definition"] == nil {
		errors = append(errors, map[string]string{"field": "workflow_definition", "message": "required"})
	} else if wd, ok := data["workflow_definition"].(map[string]any); ok {
		if wd["states"] == nil {
			errors = append(errors, map[string]string{"field": "workflow_definition.states", "message": "required"})
		}
		if wd["transitions"] == nil {
			errors = append(errors, map[string]string{"field": "workflow_definition.transitions", "message": "required"})
		}
	}

	valid := len(errors) == 0
	return jsonResult(map[string]any{"valid": valid, "errors": errors})
}

func (s *Server) validateBundle(args map[string]any) ToolResult {
	path, _ := args["path"].(string)

	// Find the cableknit binary (same binary that's running the MCP server)
	self, err := os.Executable()
	if err != nil {
		self = "cableknit"
	}

	cmdArgs := []string{"validate", "--json"}
	if path != "" {
		cmdArgs = append(cmdArgs, path)
	}

	cmd := exec.Command(self, cmdArgs...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()

	// cableknit validate returns exit code 0 for valid, non-zero for invalid
	// but --json always produces structured output either way
	if len(output) > 0 {
		// Try to parse as JSON for pretty output
		var parsed any
		if json.Unmarshal(output, &parsed) == nil {
			return jsonResult(parsed)
		}
		return textResult(string(output))
	}

	if err != nil {
		return textError(fmt.Sprintf("validate failed: %s", err.Error()))
	}

	return textResult(string(output))
}

func (s *Server) explainError(args map[string]any) ToolResult {
	msg, _ := args["error_message"].(string)
	if msg == "" {
		return textError("error_message is required")
	}

	raw := s.cache.MCPContentJSON("error_explanations")
	if raw == nil {
		return textResult(fmt.Sprintf("Error: %s\n\nUnable to load error explanations database. Check API connection.", msg))
	}

	var explanations map[string]map[string]string
	if err := json.Unmarshal(raw, &explanations); err != nil {
		return textResult(fmt.Sprintf("Error: %s\n\nFailed to parse error explanations.", msg))
	}

	msgLower := strings.ToLower(msg)
	for pattern, info := range explanations {
		patLower := strings.ToLower(pattern)
		// Support simple glob: split on .* and check all parts are present in order
		parts := strings.Split(patLower, ".*")
		matched := true
		remaining := msgLower
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			idx := strings.Index(remaining, part)
			if idx < 0 {
				matched = false
				break
			}
			remaining = remaining[idx+len(part):]
		}
		if matched {
			return jsonResult(map[string]any{
				"error":       msg,
				"explanation": info["explanation"],
				"fix":         info["fix"],
			})
		}
	}

	return jsonResult(map[string]any{
		"error":       msg,
		"explanation": "No specific explanation found for this error.",
		"fix":         "Check the relevant schema using get_*_schema tools, or run `cableknit validate` for full validation output.",
	})
}

func (s *Server) describeAutomation(args map[string]any) ToolResult {
	content, _ := args["content"].(string)
	if content == "" {
		return textError("content is required")
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return textError(fmt.Sprintf("Invalid JSON: %s", err.Error()))
	}

	// Extract workflow_definition if this is a full automation object
	wd, ok := data["workflow_definition"].(map[string]any)
	if !ok {
		// Assume the input IS the workflow_definition
		wd = data
	}

	statesRaw, _ := wd["states"].([]any)
	transitionsRaw, _ := wd["transitions"].([]any)
	if len(statesRaw) == 0 {
		return textError("No states found in workflow_definition.")
	}

	// Index states by name
	type stateInfo struct {
		Name        string
		StateType   string
		EntryAction map[string]any
	}
	states := make(map[string]stateInfo)
	var stateOrder []string
	for _, s := range statesRaw {
		sm, _ := s.(map[string]any)
		if sm == nil {
			continue
		}
		name, _ := sm["name"].(string)
		st, _ := sm["type"].(string)
		ea, _ := sm["entry_action"].(map[string]any)
		states[name] = stateInfo{Name: name, StateType: st, EntryAction: ea}
		stateOrder = append(stateOrder, name)
	}

	// Group transitions by from_state
	type transInfo struct {
		To        string
		Condition map[string]any
	}
	transMap := make(map[string][]transInfo)
	for _, t := range transitionsRaw {
		tm, _ := t.(map[string]any)
		if tm == nil {
			continue
		}
		from, _ := tm["from_state"].(string)
		to, _ := tm["to_state"].(string)
		cond, _ := tm["condition"].(map[string]any)
		transMap[from] = append(transMap[from], transInfo{To: to, Condition: cond})
	}

	// Track which states are only reached as transition targets
	transitionTargets := make(map[string]bool)
	for _, ts := range transMap {
		for _, tr := range ts {
			transitionTargets[tr.To] = true
		}
	}

	var lines []string
	step := 1

	for _, name := range stateOrder {
		st := states[name]

		// Skip states that are only reached via transitions (they're described inline)
		if transitionTargets[name] && st.StateType != "initial" && len(transMap[name]) == 0 {
			continue
		}

		// Describe state and entry action
		typeTag := ""
		if st.StateType != "" {
			typeTag = fmt.Sprintf(" (%s)", st.StateType)
		}

		actionDesc := describeAction(st.EntryAction)
		lines = append(lines, fmt.Sprintf("%d. [%s]%s — %s", step, name, typeTag, actionDesc))
		step++

		// Describe transitions from this state
		for _, tr := range transMap[name] {
			condDesc := describeCondition(tr.Condition)
			targetSt := states[tr.To]
			targetTag := ""
			if targetSt.StateType != "" {
				targetTag = fmt.Sprintf(" (%s)", targetSt.StateType)
			}
			targetAction := describeAction(targetSt.EntryAction)
			lines = append(lines, fmt.Sprintf("   %s → [%s]%s — %s", condDesc, tr.To, targetTag, targetAction))
		}
	}

	result := "Flow Description:\n" + strings.Join(lines, "\n")

	// Prepend automation metadata if available
	if name, ok := data["name"].(string); ok && name != "" {
		header := fmt.Sprintf("Automation: %s", name)
		if trigger, ok := data["trigger_type"].(string); ok {
			header += fmt.Sprintf(" (trigger: %s)", trigger)
		}
		result = header + "\n\n" + result
	}

	return textResult(result)
}

func describeAction(action map[string]any) string {
	if action == nil {
		return "no action"
	}
	actionType, _ := action["type"].(string)
	switch actionType {
	case "ai_assess":
		prompt, _ := action["prompt"].(string)
		if prompt != "" {
			if len(prompt) > 80 {
				prompt = prompt[:80] + "..."
			}
			return fmt.Sprintf("AI: \"%s\"", prompt)
		}
		return "AI assessment"
	case "notify":
		channel, _ := action["channel"].(string)
		addrs, _ := action["to_addresses"].([]any)
		subject, _ := action["subject"].(string)
		to := ""
		if len(addrs) > 0 {
			if s, ok := addrs[0].(string); ok {
				to = s
			}
		}
		desc := fmt.Sprintf("Notify via %s", channel)
		if to != "" {
			desc += fmt.Sprintf(" to %s", to)
		}
		if subject != "" {
			if len(subject) > 50 {
				subject = subject[:50] + "..."
			}
			desc += fmt.Sprintf(": \"%s\"", subject)
		}
		return desc
	case "request_decision":
		title, _ := action["title"].(string)
		opts := ""
		if options, ok := action["decision_options"].([]any); ok {
			var optStrs []string
			for _, o := range options {
				if s, ok := o.(string); ok {
					optStrs = append(optStrs, s)
				}
			}
			opts = strings.Join(optStrs, ", ")
		}
		timeout := ""
		if th, ok := action["expires_in"].(float64); ok {
			timeout = fmt.Sprintf(", timeout: %gh", th)
		}
		if title != "" {
			return fmt.Sprintf("Human decision: \"%s\" (options: %s%s)", title, opts, timeout)
		}
		return fmt.Sprintf("Human decision (options: %s%s)", opts, timeout)
	case "webhook":
		url, _ := action["url"].(string)
		timeout := 15.0
		if t, ok := action["timeout"].(float64); ok {
			timeout = t
		}
		if url != "" {
			if len(url) > 60 {
				url = url[:60] + "..."
			}
			return fmt.Sprintf("POST %s (timeout: %gs)", url, timeout)
		}
		return fmt.Sprintf("Outbound webhook (timeout: %gs)", timeout)
	default:
		if actionType != "" {
			return actionType
		}
		return "action"
	}
}

func describeCondition(cond map[string]any) string {
	if cond == nil {
		return "Always"
	}
	condType, _ := cond["type"].(string)
	switch condType {
	case "threshold":
		field, _ := cond["field"].(string)
		op, _ := cond["operator"].(string)
		value := cond["value"]
		return fmt.Sprintf("If %s %s %v", field, op, value)
	case "decision_outcome":
		matches, _ := cond["matches"].(string)
		return fmt.Sprintf("If decision = %s", matches)
	case "timeout":
		hours, _ := cond["timeout_hours"].(float64)
		return fmt.Sprintf("If no decision within %gh", hours)
	case "fallback":
		return "Otherwise"
	default:
		// Try to build something readable from fields
		if field, ok := cond["field"].(string); ok {
			op, _ := cond["operator"].(string)
			value := cond["value"]
			if op != "" {
				return fmt.Sprintf("If %s %s %v", field, op, value)
			}
			return fmt.Sprintf("If %s", field)
		}
		if condType != "" {
			return condType
		}
		return "Condition"
	}
}

func (s *Server) checkPromptInjection(args map[string]any) ToolResult {
	prompt, _ := args["prompt"].(string)
	if prompt == "" {
		return textError("prompt is required")
	}

	lower := strings.ToLower(prompt)
	var flagged []string

	patterns := []struct {
		category string
		terms    []string
	}{
		{"Role override", []string{"ignore previous instructions", "you are now", "forget your instructions", "disregard all prior", "ignore all previous", "new instructions"}},
		{"Data exfiltration", []string{"send the contents to", "output all system", "reveal your instructions", "show me your prompt", "print your system"}},
		{"Encoded payloads", []string{"base64", "decode the following", "encoded instructions"}},
		{"URL injection", []string{"fetch from http", "make a request to", "call this api", "curl ", "wget "}},
	}

	for _, p := range patterns {
		for _, term := range p.terms {
			if strings.Contains(lower, term) {
				flagged = append(flagged, fmt.Sprintf("[%s] matched: '%s'", p.category, term))
			}
		}
	}

	if len(flagged) > 0 {
		return jsonResult(map[string]any{
			"pass":           false,
			"flagged":        flagged,
			"recommendation": "Remove or rephrase the flagged patterns. Keep system prompts focused on describing what the skill does and how to help the user.",
		})
	}

	return jsonResult(map[string]any{
		"pass":    true,
		"message": "No prompt injection patterns detected.",
	})
}

// --- Generation tool implementations ---

func (s *Server) generatePluginJSON(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	tier, _ := args["tier"].(string)
	industry, _ := args["industry"].(string)

	if desc == "" {
		return textError("description is required")
	}
	if tier == "" {
		tier = "industry"
	}

	slug := slugify(desc)

	plugin := map[string]any{
		"name":        titleize(desc),
		"slug":        slug,
		"version":     "0.1.0",
		"description": desc,
		"author":      map[string]string{"name": "REPLACE_WITH_YOUR_NAME"},
		"tier":        tier,
		"visibility":  "private",
		"pricing": map[string]any{
			"model":       "monthly",
			"price_cents": 50000,
		},
		"settings_schema": []any{},
		"platform_tools":  []any{},
	}

	if industry != "" {
		plugin["industry"] = industry
	}

	// Apply generation hints if available
	raw := s.cache.MCPContentJSON("generation_hints")
	if raw != nil {
		var hints map[string]any
		if json.Unmarshal(raw, &hints) == nil {
			if industry != "" {
				if perIndustry, ok := hints["per_industry"].(map[string]any); ok {
					if industryHints, ok := perIndustry[industry].(map[string]any); ok {
						if tools, ok := industryHints["common_tools"].([]any); ok {
							plugin["platform_tools"] = tools
						}
						if settings, ok := industryHints["suggested_settings"].([]any); ok {
							var schema []map[string]any
							for _, s := range settings {
								if key, ok := s.(string); ok {
									schema = append(schema, map[string]any{
										"key":      key,
										"label":    titleize(key),
										"type":     "string",
										"required": false,
									})
								}
							}
							if len(schema) > 0 {
								plugin["settings_schema"] = schema
							}
						}
					}
				}
			}
		}
	}

	pretty, _ := json.MarshalIndent(plugin, "", "  ")
	return jsonResult(map[string]any{
		"content":      json.RawMessage(pretty),
		"instructions": "Write this to plugin.json at the root of your bundle directory. Replace REPLACE_WITH_YOUR_NAME with the publisher name. Adjust price_cents to reflect value delivered (must be above cost floor).",
	})
}

func (s *Server) generateAutomation(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	triggerType, _ := args["trigger_type"].(string)
	pluginSlug, _ := args["plugin_slug"].(string)

	if desc == "" || triggerType == "" || pluginSlug == "" {
		return textError("description, trigger_type, and plugin_slug are required")
	}

	slug := slugify(desc)

	auto := map[string]any{
		"name":         titleize(desc),
		"slug":         slug,
		"category":     "intake",
		"trigger_type": triggerType,
		"description":  desc,
		"workflow_definition": map[string]any{
			"states": []map[string]any{
				{
					"name": "assess",
					"type": "initial",
					"entry_action": map[string]any{
						"type":       "ai_assess",
						"prompt":     fmt.Sprintf("Analyze the incoming data for: %s. Extract key fields and summarize.", desc),
						"output_key": "assessment",
					},
				},
				{
					"name": "complete",
					"type": "terminal",
					"entry_action": map[string]any{
						"type":         "notify",
						"channel":      "email",
						"to_addresses": []string{"REPLACE@example.com"},
						"subject":      fmt.Sprintf("%s — completed", titleize(desc)),
						"body":         "Automation completed. See assessment: {{assessment}}",
					},
				},
			},
			"transitions": []map[string]any{
				{
					"from_state": "assess",
					"to_state":   "complete",
					"priority":   1,
					"condition":  map[string]any{"type": "fallback"},
				},
			},
		},
	}

	if triggerType == "inbound_email" {
		auto["inbound_email_slug"] = slug
	}

	pretty, _ := json.MarshalIndent(auto, "", "  ")
	return jsonResult(map[string]any{
		"content":      json.RawMessage(pretty),
		"instructions": fmt.Sprintf("Write this to automations/%s.json. Customize the workflow states, transitions, and conditions for your use case. Add threshold conditions for routing and request_decision for human approval gates.", slug),
	})
}

func (s *Server) generateBlueprint(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	artifactType, _ := args["artifact_type"].(string)
	pluginSlug, _ := args["plugin_slug"].(string)

	if desc == "" || artifactType == "" || pluginSlug == "" {
		return textError("description, artifact_type, and plugin_slug are required")
	}

	slug := slugify(desc)

	bp := map[string]any{
		"name":          titleize(desc),
		"slug":          slug,
		"artifact_type": artifactType,
		"description":   desc,
		"content_schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		"default_content": map[string]any{},
		"instructions":    fmt.Sprintf("Generate a %s for: %s. Use the content_schema to structure the output.", artifactType, desc),
	}

	pretty, _ := json.MarshalIndent(bp, "", "  ")
	return jsonResult(map[string]any{
		"content":      json.RawMessage(pretty),
		"instructions": fmt.Sprintf("Write this to blueprints/%s.json. Define content_schema properties and default_content for your artifact.", slug),
	})
}

func (s *Server) generateDataSource(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	sourceType, _ := args["source_type"].(string)
	pluginSlug, _ := args["plugin_slug"].(string)

	if desc == "" || sourceType == "" || pluginSlug == "" {
		return textError("description, source_type, and plugin_slug are required")
	}

	slug := slugify(desc)
	sourceConfig := map[string]any{}

	switch sourceType {
	case "connector":
		sourceConfig["connector"] = "REPLACE_WITH_CONNECTOR_SLUG"
	case "static":
		sourceConfig["data"] = map[string]any{}
	}

	ds := map[string]any{
		"name":        titleize(desc),
		"slug":        slug,
		"description": desc,
		"parameters":  []map[string]any{{"name": "query", "type": "string", "description": "Search query"}},
		"source": map[string]any{
			"type":   sourceType,
			"config": sourceConfig,
		},
	}

	pretty, _ := json.MarshalIndent(ds, "", "  ")
	return jsonResult(map[string]any{
		"content":      json.RawMessage(pretty),
		"instructions": fmt.Sprintf("Write this to tools/%s.json. Customize parameters and source config for your data source.", slug),
	})
}

func (s *Server) generateSkill(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	pluginSlug, _ := args["plugin_slug"].(string)

	if desc == "" || pluginSlug == "" {
		return textError("description and plugin_slug are required")
	}

	slug := slugify(desc)

	skill := map[string]any{
		"name":          titleize(desc),
		"slug":          slug,
		"action_type":   "chat",
		"system_prompt": fmt.Sprintf("You are a helpful assistant for %s. Help users understand their data, answer questions, and provide actionable recommendations. Reference the company's configuration and recent activity when relevant.", desc),
	}

	pretty, _ := json.MarshalIndent(skill, "", "  ")
	return jsonResult(map[string]any{
		"content":      json.RawMessage(pretty),
		"instructions": fmt.Sprintf("Write this to skills/%s.json. Customize the system_prompt with domain-specific knowledge and behavior. Avoid prompt injection patterns (role overrides, data exfiltration, URL injection).", slug),
	})
}

func (s *Server) generateDocPage(args map[string]any) ToolResult {
	title, _ := args["title"].(string)
	topic, _ := args["topic"].(string)
	pluginSlug, _ := args["plugin_slug"].(string)

	if title == "" || topic == "" || pluginSlug == "" {
		return textError("title, topic, and plugin_slug are required")
	}

	slug := slugify(title)

	content := fmt.Sprintf(`---
title: %s
slug: %s
category: getting-started
---

# %s

%s

## Setup

1. Install the plugin from the CableKnit marketplace
2. Configure required settings in the dashboard
3. Follow the instructions below to get started

## Usage

Describe how to use this feature here.

## Configuration

List any configuration options and what they do.
`, title, slug, title, topic)

	return jsonResult(map[string]any{
		"content":      content,
		"instructions": fmt.Sprintf("Write this to docs/%s.md. Expand the sections with real content for your plugin.", slug),
	})
}

func (s *Server) scaffoldBundle(args map[string]any) ToolResult {
	desc, _ := args["description"].(string)
	name, _ := args["name"].(string)
	industry, _ := args["industry"].(string)

	if desc == "" || name == "" {
		return textError("description and name are required")
	}

	slug := slugify(name)

	// Generate all the files
	pluginArgs := map[string]any{"description": desc, "tier": "industry"}
	if industry != "" {
		pluginArgs["industry"] = industry
	}
	pluginResult := s.generatePluginJSON(pluginArgs)

	autoResult := s.generateAutomation(map[string]any{
		"description":  fmt.Sprintf("Main workflow for %s", desc),
		"trigger_type": "event",
		"plugin_slug":  slug,
	})

	skillResult := s.generateSkill(map[string]any{
		"description": desc,
		"plugin_slug": slug,
	})

	docResult := s.generateDocPage(map[string]any{
		"title":       "Getting Started",
		"topic":       fmt.Sprintf("How to set up and use %s", name),
		"plugin_slug": slug,
	})

	// Extract content from results
	files := map[string]any{
		"directory": slug,
		"files": map[string]any{
			"plugin.json":                       extractContent(pluginResult),
			"automations/main-workflow.json":     extractContent(autoResult),
			"skills/" + slugify(desc) + ".json":  extractContent(skillResult),
			"docs/getting-started.md":            extractContent(docResult),
		},
		"directories_to_create": []string{
			slug,
			slug + "/automations",
			slug + "/skills",
			slug + "/blueprints",
			slug + "/tools",
			slug + "/docs",
			slug + "/images",
		},
		"instructions": fmt.Sprintf("Create the directory structure and write each file. Then add a 512x512 PNG icon at %s/images/icon.png. Run `cableknit validate ./%s` to check the bundle.", slug, slug),
	}

	return jsonResult(files)
}

// --- Helpers ---

func emptySchema() any {
	return map[string]any{"type": "object"}
}

func textResult(text string) ToolResult {
	return ToolResult{Content: []Content{{Type: "text", Text: text}}}
}

func textError(msg string) ToolResult {
	return ToolResult{Content: []Content{{Type: "text", Text: msg}}, IsError: true}
}

func jsonResult(v any) ToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return textError(fmt.Sprintf("JSON marshal error: %s", err))
	}
	return textResult(string(data))
}

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var result []byte
	prevDash := false
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result = append(result, byte(c))
			prevDash = false
		} else if !prevDash && len(result) > 0 {
			result = append(result, '-')
			prevDash = true
		}
	}
	// Trim trailing dash
	if len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	// Truncate
	if len(result) > 40 {
		result = result[:40]
		if result[len(result)-1] == '-' {
			result = result[:len(result)-1]
		}
	}
	return string(result)
}

func titleize(s string) string {
	if len(s) == 0 {
		return s
	}
	// Simple titleize: capitalize first letter of each word
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	result := strings.Join(words, " ")
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}

func extractContent(r ToolResult) any {
	if len(r.Content) == 0 {
		return nil
	}
	text := r.Content[0].Text

	// Try to parse as JSON with a "content" field
	var wrapper map[string]json.RawMessage
	if json.Unmarshal([]byte(text), &wrapper) == nil {
		if content, ok := wrapper["content"]; ok {
			// Try to parse the content as JSON
			var parsed any
			if json.Unmarshal(content, &parsed) == nil {
				return parsed
			}
			// It might be a string (for markdown)
			var str string
			if json.Unmarshal(content, &str) == nil {
				return str
			}
		}
	}
	return text
}
