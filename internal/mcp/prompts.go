package mcp

import (
	"fmt"
	"strings"
)

func allPrompts() []Prompt {
	return []Prompt{
		{
			Name:        "build-plugin",
			Description: "Walk through building a complete CableKnit plugin from scratch — plugin.json, automations, skills, docs.",
			Arguments: []PromptArgument{
				{Name: "description", Description: "What the plugin should do.", Required: true},
			},
		},
		{
			Name:        "debug-validation",
			Description: "Debug CableKnit bundle validation errors — explains each error and suggests fixes.",
			Arguments: []PromptArgument{
				{Name: "error_output", Description: "Paste the output from `cableknit validate`.", Required: true},
			},
		},
		{
			Name:        "add-automation",
			Description: "Add a new automation to an existing CableKnit plugin.",
			Arguments: []PromptArgument{
				{Name: "description", Description: "What the automation should do.", Required: true},
				{Name: "trigger_type", Description: "What starts the automation: inbound_email, webhook, schedule, or event.", Required: true},
			},
		},
		{
			Name:        "review-before-submit",
			Description: "Review your plugin bundle against app guidelines before submitting for review.",
			Arguments:   []PromptArgument{},
		},
	}
}

func (s *Server) getPrompt(params PromptGetParams) (*PromptGetResult, error) {
	// Map prompt name to MCP content key
	keyMap := map[string]string{
		"build-plugin":         "prompt_build_plugin",
		"debug-validation":     "prompt_debug_validation",
		"add-automation":       "prompt_add_automation",
		"review-before-submit": "prompt_review_before_submit",
	}

	contentKey, ok := keyMap[params.Name]
	if !ok {
		return nil, fmt.Errorf("unknown prompt: %s", params.Name)
	}

	template := s.cache.MCPContent(contentKey)
	if template == "" {
		return nil, fmt.Errorf("prompt template not available: %s", params.Name)
	}

	// Substitute arguments into template
	text := template
	for k, v := range params.Arguments {
		if str, ok := v.(string); ok {
			text = strings.ReplaceAll(text, "{{"+k+"}}", str)
		}
	}

	return &PromptGetResult{
		Description: promptDescription(params.Name),
		Messages: []PromptMessage{
			{
				Role:    "user",
				Content: Content{Type: "text", Text: text},
			},
		},
	}, nil
}

func promptDescription(name string) string {
	switch name {
	case "build-plugin":
		return "Build a CableKnit plugin from scratch"
	case "debug-validation":
		return "Debug validation errors"
	case "add-automation":
		return "Add an automation to an existing plugin"
	case "review-before-submit":
		return "Review plugin before submission"
	default:
		return ""
	}
}
