package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MockClient wraps Client and returns fake data with realistic delays.
type MockClient struct {
	*Client
}

func NewMockClient() *MockClient {
	return &MockClient{Client: NewClient()}
}

func (mc *MockClient) JSON(method, path string, body any, result any) error {
	switch {
	case method == "GET" && path == "/api/v1/cli/manifest":
		// Manifest always fetches from real API, even in demo mode
		return mc.Client.JSON(method, path, body, result)

	case method == "POST" && path == "/api/v1/cli/sessions":
		time.Sleep(800 * time.Millisecond)
		DemoLoggedIn = true
		return mockUnmarshal(result, LoginResponse{
			Token: "ck_mock_token_abc123",
			User: User{
				Name:            "Demo User",
				Email:           "demo@cableknit.ai",
				PublisherStatus: "active",
			},
		})

	case method == "DELETE" && path == "/api/v1/cli/sessions":
		time.Sleep(200 * time.Millisecond)
		return nil

	case method == "GET" && path == "/api/v1/cli/me":
		time.Sleep(400 * time.Millisecond)
		return mockUnmarshal(result, User{
			Name:            "Demo User",
			Email:           "demo@cableknit.ai",
			PublisherStatus: "active",
		})

	case method == "GET" && path == "/api/v1/cli/plugins":
		time.Sleep(600 * time.Millisecond)
		if m := GetManifest(); m != nil {
			if data := m.JSONContent("demo_data", "mock_plugins"); data != nil {
				var plugins []Plugin
				if json.Unmarshal(data, &plugins) == nil && len(plugins) > 0 {
					return mockUnmarshal(result, PluginsResponse{Plugins: plugins})
				}
			}
		}
		return mockUnmarshal(result, PluginsResponse{
			Plugins: []Plugin{
				{Name: "Slack Notifier", Slug: "slack-notifier", Version: "1.2.0", Status: "published", Visibility: "public", Installs: 1432},
				{Name: "GitHub Sync", Slug: "github-sync", Version: "2.0.1", Status: "published", Visibility: "public", Installs: 876},
				{Name: "Email Digest", Slug: "email-digest", Version: "0.9.0", Status: "draft", Visibility: "private", Installs: 0},
			},
		})

	case method == "GET" && strings.HasPrefix(path, "/api/v1/cli/runs"):
		time.Sleep(500 * time.Millisecond)
		if m := GetManifest(); m != nil {
			if data := m.JSONContent("demo_data", "mock_runs"); data != nil {
				var rawRuns []struct {
					ShortToken string `json:"short_token"`
					Automation string `json:"automation"`
					Status     string `json:"status"`
					State      string `json:"state"`
					MinutesAgo int    `json:"minutes_ago"`
				}
				if json.Unmarshal(data, &rawRuns) == nil && len(rawRuns) > 0 {
					now := time.Now()
					var runs []Run
					for _, r := range rawRuns {
						runs = append(runs, Run{
							ShortToken: r.ShortToken,
							Automation: r.Automation,
							Status:     r.Status,
							State:      r.State,
							CreatedAt:  now.Add(-time.Duration(r.MinutesAgo) * time.Minute),
						})
					}
					return mockUnmarshal(result, RunsResponse{Runs: runs})
				}
			}
		}
		now := time.Now()
		return mockUnmarshal(result, RunsResponse{
			Runs: []Run{
				{ShortToken: "RUN-7F3A", Automation: "deploy-staging", Status: "running", State: "executing step 3/5", CreatedAt: now.Add(-2 * time.Minute)},
				{ShortToken: "RUN-4B2C", Automation: "nightly-sync", Status: "completed", State: "done", CreatedAt: now.Add(-1 * time.Hour)},
				{ShortToken: "RUN-9D1E", Automation: "data-import", Status: "paused_for_decision", State: "awaiting approval", CreatedAt: now.Add(-15 * time.Minute)},
			},
		})

	case method == "GET" && path == "/api/v1/cli/platform_tools":
		time.Sleep(500 * time.Millisecond)
		if m := GetManifest(); m != nil {
			if data := m.JSONContent("demo_data", "mock_platform_tools"); data != nil {
				var tools []PlatformTool
				if json.Unmarshal(data, &tools) == nil && len(tools) > 0 {
					return mockUnmarshal(result, PlatformToolsResponse{PlatformTools: tools})
				}
			}
		}
		return mockUnmarshal(result, PlatformToolsResponse{
			PlatformTools: []PlatformTool{
				{Name: "Lookup Employees", Slug: "lookup-employees", Description: "Search company employee directory", Category: "platform"},
				{Name: "Fetch Briefing", Slug: "fetch-briefing", Description: "Retrieve daily operational briefing", Category: "platform"},
				{Name: "Update Cells", Slug: "update-cells", Description: "Write values to spreadsheet cells", Category: "contextual"},
			},
		})

	case method == "GET" && strings.HasSuffix(path, "/logs"):
		time.Sleep(500 * time.Millisecond)
		now := time.Now()
		return mockUnmarshal(result, LogsResponse{
			Logs: []LogEntry{
				{EventType: "sandbox_execution", Severity: "info", Metadata: map[string]interface{}{"tool_name": "execute_slack_notifier_code", "duration_ms": 142}, OccurredAt: now.Add(-5 * time.Minute)},
				{EventType: "sandbox_error", Severity: "error", Metadata: map[string]interface{}{"tool_name": "execute_slack_notifier_code", "error": "Timeout after 5s"}, OccurredAt: now.Add(-12 * time.Minute)},
				{EventType: "automation_started", Severity: "info", Metadata: map[string]interface{}{"automation_name": "daily-sync"}, OccurredAt: now.Add(-1 * time.Hour)},
				{EventType: "automation_completed", Severity: "info", Metadata: map[string]interface{}{"automation_name": "daily-sync"}, OccurredAt: now.Add(-59 * time.Minute)},
				{EventType: "bundle_pushed", Severity: "info", Metadata: map[string]interface{}{"version": "1.2.0"}, OccurredAt: now.Add(-3 * time.Hour)},
			},
		})

	case method == "GET" && strings.HasSuffix(path, "/metrics"):
		time.Sleep(400 * time.Millisecond)
		return mockUnmarshal(result, MetricsResponse{
			ToolCalls24h:         87,
			ToolCalls7d:          543,
			ToolCalls30d:         2104,
			Errors24h:            3,
			ErrorRate7d:          1.24,
			AutomationRuns7d:     28,
			AutomationFailures7d: 2,
			ActiveInstalls:       14,
			AvgSandboxDurationMs: 156.3,
		})

	case method == "GET" && strings.HasSuffix(path, "/errors"):
		time.Sleep(400 * time.Millisecond)
		now := time.Now()
		return mockUnmarshal(result, ErrorsResponse{
			Errors: []LogEntry{
				{EventType: "sandbox_error", Severity: "error", Metadata: map[string]interface{}{"tool_name": "execute_slack_notifier_code", "error": "Timeout after 5s"}, OccurredAt: now.Add(-12 * time.Minute)},
				{EventType: "automation_failed", Severity: "error", Metadata: map[string]interface{}{"automation_name": "daily-sync", "error": "API rate limit exceeded"}, OccurredAt: now.Add(-2 * time.Hour)},
			},
		})

	case method == "GET" && strings.HasSuffix(path, "/versions"):
		time.Sleep(400 * time.Millisecond)
		now := time.Now()
		return mockUnmarshal(result, VersionsResponse{
			Versions: []PluginVersion{
				{Version: "1.2.0", Notes: "Added Slack integration", PushedAt: now.Add(-3 * time.Hour), PushedBy: "Demo User"},
				{Version: "1.1.0", Notes: "Bug fixes", PushedAt: now.Add(-24 * time.Hour), PushedBy: "Demo User"},
				{Version: "1.0.0", Notes: "Initial release", PushedAt: now.Add(-72 * time.Hour), PushedBy: "Demo User"},
			},
		})

	case method == "POST" && strings.HasSuffix(path, "/rollback"):
		time.Sleep(1 * time.Second)
		return mockUnmarshal(result, RollbackResponse{
			Success: true,
			Version: "1.1.0",
			Message: "Rolled back to v1.1.0",
		})

	case method == "GET" && path == "/api/v1/stats":
		time.Sleep(300 * time.Millisecond)
		return mockUnmarshal(result, StatsResponse{
			TotalPublishedApps:  12,
			SubscribedAppsCount: 3,
			PayoutYTDCents:      48500,
		})
	}

	return fmt.Errorf("mock: unhandled %s %s", method, path)
}

func (mc *MockClient) Multipart(path string, fieldName string, fileName string, r io.Reader, result any) error {
	return mc.MultipartWithProgress(path, fieldName, fileName, r, 0, nil, result)
}

func (mc *MockClient) MultipartWithProgress(path string, fieldName string, fileName string, r io.Reader, size int64, onProgress func(int64), result any) error {
	switch {
	case strings.Contains(path, "validate"):
		time.Sleep(1200 * time.Millisecond)
		return mockUnmarshal(result, ValidateResponse{
			Valid:       true,
			PluginName:  "My Plugin",
			Slug:        "my-plugin",
			Version:     "1.0.0",
			Skills:      3,
			Automations: 2,
			Docs:        5,
		})

	case strings.Contains(path, "push"):
		time.Sleep(2 * time.Second)
		return mockUnmarshal(result, PushResponse{
			PluginName: "My Plugin",
			Slug:       "my-plugin",
			Version:    "1.0.0",
			Updated:    false,
			Message:    "Plugin created successfully",
		})
	}

	return fmt.Errorf("mock: unhandled multipart %s", path)
}

func (mc *MockClient) SSE(path string) (*http.Response, error) {
	events := defaultSSEEvents()

	// Try manifest data
	if m := GetManifest(); m != nil {
		if data := m.JSONContent("demo_data", "mock_sse_events"); data != nil {
			var rawEvents []struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}
			if json.Unmarshal(data, &rawEvents) == nil && len(rawEvents) > 0 {
				events = make([]RunEvent, len(rawEvents))
				now := time.Now()
				for i, e := range rawEvents {
					events[i] = RunEvent{
						Type:      e.Type,
						Message:   e.Message,
						Timestamp: now.Add(time.Duration(i) * time.Second),
					}
				}
			}
		}
	}

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		for i, ev := range events {
			if i > 0 {
				time.Sleep(900 * time.Millisecond)
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(pw, "data: %s\n\n", data)
		}
	}()

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(pr),
	}
	return resp, nil
}

func defaultSSEEvents() []RunEvent {
	now := time.Now()
	return []RunEvent{
		{Type: "log", Message: "Initializing run environment...", Timestamp: now},
		{Type: "log", Message: "Loading plugin configuration", Timestamp: now.Add(1 * time.Second)},
		{Type: "log", Message: "Connecting to external services", Timestamp: now.Add(2 * time.Second)},
		{Type: "completed", Message: "Run completed successfully", Timestamp: now.Add(3 * time.Second)},
	}
}

func mockUnmarshal(dest any, src any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

// DemoEnabled is set by the --demo flag
var DemoEnabled bool

// DemoLoggedIn tracks login state in demo mode (starts false)
var DemoLoggedIn bool

// GetClient returns a mock or real client based on demo mode.
func GetClient() interface {
	JSON(method, path string, body any, result any) error
	Multipart(path string, fieldName string, fileName string, r io.Reader, result any) error
	SSE(path string) (*http.Response, error)
} {
	if DemoEnabled {
		return NewMockClient()
	}
	return NewClient()
}

// APIClient is the interface commands use.
type APIClient interface {
	JSON(method, path string, body any, result any) error
	Multipart(path string, fieldName string, fileName string, r io.Reader, result any) error
	SSE(path string) (*http.Response, error)
}

// Ensure both types satisfy the interface
var (
	_ APIClient = (*Client)(nil)
	_ APIClient = (*MockClient)(nil)
)

// NewAPIClient returns mock or real client.
func NewAPIClient() APIClient {
	if DemoEnabled {
		return NewMockClient()
	}
	return NewClient()
}

