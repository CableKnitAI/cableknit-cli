package api

import "time"

// Auth
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	PublisherStatus string `json:"publisher_status"`
}

// Validate / Push
type ValidationError struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Level   string `json:"level"` // "error" or "warning"
}

type ValidateResponse struct {
	Valid       bool              `json:"valid"`
	PluginName string            `json:"plugin_name"`
	Slug       string            `json:"slug"`
	Version    string            `json:"version"`
	Skills     int               `json:"skills"`
	Automations int              `json:"automations"`
	Blueprints  int              `json:"blueprints"`
	Tools      int               `json:"tools"`
	Docs       int               `json:"docs"`
	Errors     []ValidationError `json:"errors,omitempty"`
	Warnings   []ValidationError `json:"warnings,omitempty"`
}

type PushResponse struct {
	PluginName string `json:"plugin_name"`
	Slug       string `json:"slug"`
	Version    string `json:"version"`
	Updated    bool   `json:"updated"`
	Message    string `json:"message"`
	Errors     []ValidationError `json:"errors,omitempty"`
}

// Plugins
type Plugin struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	Visibility string `json:"visibility"`
	Installs   int    `json:"installs"`
}

type PluginsResponse struct {
	Plugins []Plugin `json:"plugins"`
}

// Runs
type Run struct {
	ShortToken string    `json:"short_token"`
	Automation string    `json:"automation"`
	Status     string    `json:"status"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
}

type RunsResponse struct {
	Runs []Run `json:"runs"`
}

// SSE Events
type RunEvent struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Connectors
type Connector struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Category string `json:"category"`
	AuthType string `json:"auth_type"`
	Status   string `json:"status"`
}

type ConnectorsResponse struct {
	Connectors []Connector `json:"connectors"`
}

// Platform Tools
type PlatformTool struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type PlatformToolsResponse struct {
	PlatformTools []PlatformTool `json:"platform_tools"`
}

// Stats
type StatsResponse struct {
	TotalPublishedApps   int `json:"total_published_apps"`
	SubscribedAppsCount  int `json:"subscribed_apps_count"`
	PayoutYTDCents       int `json:"payout_ytd_cents"`
}

// Observability
type LogEntry struct {
	EventType  string                 `json:"event_type"`
	Severity   string                 `json:"severity"`
	Metadata   map[string]interface{} `json:"metadata"`
	OccurredAt time.Time              `json:"occurred_at"`
}

type LogsResponse struct {
	Logs []LogEntry `json:"logs"`
}

type MetricsResponse struct {
	ToolCalls24h         int     `json:"tool_calls_24h"`
	ToolCalls7d          int     `json:"tool_calls_7d"`
	ToolCalls30d         int     `json:"tool_calls_30d"`
	Errors24h            int     `json:"errors_24h"`
	ErrorRate7d          float64 `json:"error_rate_7d"`
	AutomationRuns7d     int     `json:"automation_runs_7d"`
	AutomationFailures7d int     `json:"automation_failures_7d"`
	ActiveInstalls       int     `json:"active_installs"`
	AvgSandboxDurationMs float64 `json:"avg_sandbox_duration_ms"`
}

type ErrorsResponse struct {
	Errors []LogEntry `json:"errors"`
}

// API Errors
type APIError struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// Versions
type PluginVersion struct {
	Version  string `json:"version"`
	Notes    string `json:"notes"`
	PushedAt time.Time `json:"pushed_at"`
	PushedBy string `json:"pushed_by"`
}

type VersionsResponse struct {
	Versions []PluginVersion `json:"versions"`
}

type RollbackResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version"`
	Message string `json:"message"`
}
