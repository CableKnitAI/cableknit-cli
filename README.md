# CableKnit CLI

Command-line tool for managing CableKnit plugins and automation runs.

## Install

### Homebrew

```bash
brew install jessewaites/cableknit/cableknit
```

### Direct download

Download the latest release from [GitHub Releases](https://github.com/jessewaites/cableknit-cli/releases).

### Go install

```bash
go install github.com/jessewaites/cableknit-cli@latest
```

## Usage

### Auth

```bash
cableknit login          # interactive login
cableknit whoami         # show current user
cableknit logout         # log out
```

### Plugins

```bash
cableknit validate [path]      # validate a plugin bundle
cableknit push [path]          # push a plugin bundle
cableknit plugins list         # list your plugins
cableknit plugins list --json  # JSON output
```

Bundles can include skills, automations, data source tools, docs, images, and artifact blueprints.
See the [CLI guide](docs/cli-guide.mdx) for the full bundle format reference.

### Data source tools

Place JSON files in `tools/` to expose queryable data to the AI. Three source types are supported:

**`data_store`** — reads from the plugin's key-value store:

```json
{
  "name": "formulary",
  "slug": "formulary",
  "description": "Returns medication formulary entries. Pass an NDC code to look up one.",
  "parameters": [
    { "name": "ndc_code", "type": "string", "description": "NDC code", "required": false }
  ],
  "source": {
    "type": "data_store",
    "config": {
      "key_prefix": "formulary:",
      "single_key_template": "formulary:{{ndc_code}}"
    }
  }
}
```

**`connector`** — fetches from a connected external app:

```json
{
  "name": "projects",
  "slug": "projects",
  "description": "Returns project data from Procore.",
  "parameters": [
    { "name": "status", "type": "string", "description": "Filter by status", "required": false }
  ],
  "source": {
    "type": "connector",
    "config": {
      "connector": "procore",
      "resources": "projects",
      "filter_mapping": { "status": "{{status}}" }
    }
  }
}
```

**`static`** — a lookup table shipped in the bundle (max 64KB):

```json
{
  "name": "severity_levels",
  "slug": "severity-levels",
  "description": "OSHA severity level definitions.",
  "source": {
    "type": "static",
    "config": {
      "data": [
        { "level": 1, "name": "Imminent Danger", "response_hours": 0 },
        { "level": 2, "name": "Serious", "response_hours": 24 }
      ]
    }
  }
}
```

At runtime the AI writes Ruby code against these sources in a sandboxed Enclave VM. Max 10 tools per bundle. `cableknit generate` scaffolds two sample tools to get you started.

### Runs

```bash
cableknit runs list                    # list runs
cableknit runs list --status running   # filter by status
cableknit runs list --limit 10         # limit results
cableknit runs tail <run-id>           # stream run logs
cableknit runs tail <run-id> --no-tui  # plain text output
```

### Global flags

```
--api-url    override API base URL
--no-color   disable color output
--debug      enable debug logging
--json       output as JSON
```

## Dev setup

```bash
git clone https://github.com/jessewaites/cableknit-cli.git
cd cableknit-cli
go build .
./cableknit version
```

### Run with version info

```bash
go run -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD)" .
```
