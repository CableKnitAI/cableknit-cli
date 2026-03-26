package cmd

import (
	"os"

	"github.com/jessewaites/cableknit-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio transport)",
	Long:  "Starts a Model Context Protocol server over stdio for AI-powered editor integrations (Claude Code, Cursor, Windsurf, Copilot).",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mcp.NewServer(buildVersion)
		return s.Run(os.Stdin, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func mcpSetupContent() string {
	return `CableKnit includes a built-in MCP (Model Context Protocol) server that gives
AI-powered editors deep knowledge of the CableKnit platform. Your editor's AI
assistant can look up schemas, generate valid plugin code, validate bundles,
explain errors, and walk you through building a complete plugin — all without
leaving your editor.

Works with: Claude Code, Cursor, Windsurf, Copilot, and any MCP-compatible editor.


How It Works

The MCP server runs as a background process that your editor talks to
automatically. You never interact with it directly — your editor's AI
assistant uses it behind the scenes to give you CableKnit-aware help.


Setup

Auto-install (recommended):

  cableknit mcp install

This detects your installed editors (Claude Code, Cursor, Windsurf) and
configures them automatically. Or target a specific editor:

  cableknit mcp install claude
  cableknit mcp install cursor
  cableknit mcp install windsurf

That's it. Restart your editor and the AI assistant will have access to all
CableKnit tools.

Manual setup (if needed):

  Claude Code:   claude mcp add cableknit -- cableknit mcp

  Cursor / Windsurf:  Add to .cursor/mcp.json or .windsurf/mcp.json:

    {
      "mcpServers": {
        "cableknit": {
          "command": "cableknit",
          "args": ["mcp"]
        }
      }
    }


What Your AI Assistant Can Do

  Reference (14 tools)
    - Look up plugin.json, automation, blueprint, data source, skill, and docs schemas
    - List platform tools, connectors, industries, and categories
    - Show bundle size/count limits
    - Show app review guidelines and CLI docs
    - Check auth status

  Validation (4 tools)
    - Validate plugin.json and automation templates against schemas
    - Explain validation errors with fix suggestions
    - Pre-check system prompts for injection patterns

  Generation (7 tools)
    - Generate plugin.json, automations, blueprints, data sources, skills, doc pages
    - Scaffold a complete bundle from a description

  Prompt Templates (4 prompts)
    - "Build a plugin" — walks through the full process
    - "Debug validation" — explains errors from cableknit validate
    - "Add an automation" — generates a new automation for an existing plugin
    - "Review before submit" — checks against app guidelines


No Auth Required

All MCP tools work without logging in. Schemas, generation, and validation
are fully available. The whoami tool will show you're unauthenticated and
suggest applying for a developer account.


CLI Command

You can also run the MCP server directly:

  cableknit mcp

This starts the stdio server. Normally your editor handles this automatically.
`
}
