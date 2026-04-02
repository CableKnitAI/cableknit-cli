package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cableknitai/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var mcpInstallCmd = &cobra.Command{
	Use:   "install [editor]",
	Short: "Install MCP server config into your editor",
	Long:  "Automatically configures the CableKnit MCP server for Claude Code, Cursor, or Windsurf.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := ""
		if len(args) > 0 {
			editor = args[0]
		}

		switch editor {
		case "claude", "claude-code":
			return installClaude()
		case "cursor":
			return installJSONConfig(".cursor/mcp.json", "Cursor")
		case "windsurf":
			return installJSONConfig(".windsurf/mcp.json", "Windsurf")
		case "":
			return installAll()
		default:
			return fmt.Errorf("unknown editor: %s (supported: claude, cursor, windsurf)", editor)
		}
	},
}

func init() {
	mcpCmd.AddCommand(mcpInstallCmd)
}

func installAll() error {
	installed := 0

	// Claude Code
	if _, err := exec.LookPath("claude"); err == nil {
		if err := installClaude(); err == nil {
			installed++
		}
	}

	// Cursor
	cursorDir := filepath.Join(os.Getenv("HOME"), ".cursor")
	if info, err := os.Stat(cursorDir); err == nil && info.IsDir() {
		if err := installJSONConfig(".cursor/mcp.json", "Cursor"); err == nil {
			installed++
		}
	}

	// Windsurf
	windsurfDir := filepath.Join(os.Getenv("HOME"), ".windsurf")
	if info, err := os.Stat(windsurfDir); err == nil && info.IsDir() {
		if err := installJSONConfig(".windsurf/mcp.json", "Windsurf"); err == nil {
			installed++
		}
	}

	if installed == 0 {
		fmt.Printf("  %s No supported editors detected.\n\n", ui.SymbolWarning)
		fmt.Println("  Install manually:")
		fmt.Println()
		fmt.Println("    Claude Code:  claude mcp add cableknit -- cableknit mcp")
		fmt.Println("    Cursor:       cableknit mcp install cursor")
		fmt.Println("    Windsurf:     cableknit mcp install windsurf")
		fmt.Println()
	}

	return nil
}

func installClaude() error {
	cableknitPath, err := os.Executable()
	if err != nil {
		cableknitPath = "cableknit"
	}

	cmd := exec.Command("claude", "mcp", "add", "cableknit", "--", cableknitPath, "mcp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude mcp add failed: %w", err)
	}

	fmt.Printf("  %s CableKnit MCP server added to Claude Code\n", ui.SymbolCheck)
	return nil
}

func installJSONConfig(relPath, editorName string) error {
	home := os.Getenv("HOME")
	configPath := filepath.Join(home, relPath)

	// Ensure parent dir exists
	os.MkdirAll(filepath.Dir(configPath), 0o755)

	cableknitPath, err := os.Executable()
	if err != nil {
		cableknitPath = "cableknit"
	}

	// Read existing config or start fresh
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &config)
	}
	if config == nil {
		config = map[string]any{}
	}

	servers, _ := config["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	servers["cableknit"] = map[string]any{
		"command": cableknitPath,
		"args":    []string{"mcp"},
	}
	config["mcpServers"] = servers

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	fmt.Printf("  %s CableKnit MCP server added to %s (%s)\n", ui.SymbolCheck, editorName, configPath)
	return nil
}
