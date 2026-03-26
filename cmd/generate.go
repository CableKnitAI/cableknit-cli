package cmd

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"init", "scaffold"},
	Short:   "Generate a plugin scaffold",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ui.IsTTY() {
			return fmt.Errorf("generate requires an interactive terminal")
		}

		p := tea.NewProgram(newGenerateModel())
		m, err := p.Run()
		if err != nil {
			return err
		}
		gm := m.(generateModel)
		if gm.err != nil {
			return gm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

// --- slugify ---

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := strings.ToLower(name)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// --- generate model ---

type generateState int

const (
	generateForm generateState = iota
	generateWriting
	generateDone
	generateError
)

type generateResultMsg struct {
	slug string
	err  error
}

type generateModel struct {
	state    generateState
	form     *huh.Form
	spinner  spinner.Model
	name     string
	desc     string
	category string
	industry string
	interval string
	slug     string
	err      error
	embedded bool
}

func manifestCategories() []string {
	if m := api.GetManifest(); m != nil {
		if data := m.JSONContent("scaffold", "categories"); data != nil {
			var cats []string
			if json.Unmarshal(data, &cats) == nil && len(cats) > 0 {
				return cats
			}
		}
	}
	return []string{"intake", "processing", "notification", "integration", "analytics"}
}

type industryOption struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func manifestIndustries() []huh.Option[string] {
	options := []huh.Option[string]{huh.NewOption("None", "")}

	if m := api.GetManifest(); m != nil {
		if data := m.JSONContent("scaffold", "industries"); data != nil {
			var industries []industryOption
			if json.Unmarshal(data, &industries) == nil && len(industries) > 0 {
				for _, ind := range industries {
					options = append(options, huh.NewOption(ind.Name, ind.Slug))
				}
				return options
			}
		}
	}

	// Hardcoded fallback — common subset
	fallback := []industryOption{
		{"technology", "Technology"},
		{"healthcare", "Healthcare"},
		{"finance_and_insurance", "Finance & Insurance"},
		{"retail_and_ecommerce", "Retail & E-commerce"},
		{"manufacturing", "Manufacturing"},
		{"professional_services", "Professional Services"},
		{"construction_and_real_estate", "Construction & Real Estate"},
		{"transportation_and_logistics", "Transportation & Logistics"},
		{"education", "Education"},
		{"food_and_hospitality", "Food & Hospitality"},
	}
	for _, ind := range fallback {
		options = append(options, huh.NewOption(ind.Name, ind.Slug))
	}
	return options
}

func manifestBillingIntervals() []string {
	if m := api.GetManifest(); m != nil {
		if data := m.JSONContent("scaffold", "billing_intervals"); data != nil {
			var intervals []string
			if json.Unmarshal(data, &intervals) == nil && len(intervals) > 0 {
				return intervals
			}
		}
	}
	return []string{"monthly", "yearly"}
}

func newGenerateModel() generateModel {
	m := generateModel{}

	categories := manifestCategories()
	var catOptions []huh.Option[string]
	for _, c := range categories {
		catOptions = append(catOptions, huh.NewOption(c, c))
	}

	industryOptions := manifestIndustries()

	intervals := manifestBillingIntervals()
	var intervalOptions []huh.Option[string]
	for _, i := range intervals {
		intervalOptions = append(intervalOptions, huh.NewOption(i, i))
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Plugin Name").
				Placeholder("My Plugin").
				Value(&m.name),
			huh.NewInput().
				Title("Description").
				Placeholder("What does your plugin do?").
				Value(&m.desc),
			huh.NewSelect[string]().
				Title("Category").
				Options(catOptions...).
				Value(&m.category),
			huh.NewSelect[string]().
				Title("Industry").
				Options(industryOptions...).
				Value(&m.industry),
			huh.NewSelect[string]().
				Title("Billing Interval").
				Options(intervalOptions...).
				Value(&m.interval),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCharm))

	m.spinner = ui.NewSpinner(spinner.Dot)

	return m
}

func (m generateModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m generateModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m generateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case generateResultMsg:
		if msg.err != nil {
			m.state = generateError
			m.err = msg.err
			return m, m.done()
		}
		m.state = generateDone
		m.slug = msg.slug
		return m, m.done()
	}

	switch m.state {
	case generateForm:
		form, cmd := m.form.Update(msg)
		m.form = form.(*huh.Form)

		if m.form.State == huh.StateCompleted {
			m.slug = slugify(m.name)
			m.state = generateWriting
			return m, tea.Batch(m.spinner.Tick, m.doGenerate())
		}
		if m.form.State == huh.StateAborted {
			return m, m.done()
		}
		return m, cmd

	case generateWriting:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m generateModel) View() tea.View {
	var s string

	switch m.state {
	case generateForm:
		s = "\n" + m.form.View() + "\n"

	case generateWriting:
		s = "\n  " + m.spinner.View() + " Generating scaffold...\n\n"

	case generateDone:
		content := fmt.Sprintf(
			"%s  %s\n%s  ./%s/",
			lipgloss.NewStyle().Bold(true).Render("Plugin:"), m.name,
			lipgloss.NewStyle().Bold(true).Render("Created:"), m.slug,
		)
		s = "\n" + ui.SuccessBox.Render(content) + "\n\n"

	case generateError:
		s = "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
	}

	return tea.NewView(s)
}

func (m generateModel) doGenerate() tea.Cmd {
	return func() tea.Msg {
		slug := slugify(m.name)

		dirs := []string{
			slug,
			filepath.Join(slug, "skills"),
			filepath.Join(slug, "automations"),
			filepath.Join(slug, "blueprints"),
			filepath.Join(slug, "tools"),
			filepath.Join(slug, "docs"),
			filepath.Join(slug, "images"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0o755); err != nil {
				return generateResultMsg{err: fmt.Errorf("mkdir %s: %w", d, err)}
			}
		}

		// plugin.json — use manifest defaults if available
		defaults := map[string]any{
			"version":         "0.1.0",
			"price_cents":     50000,
			"settings_schema": []any{},
		}
		if mf := api.GetManifest(); mf != nil {
			if data := mf.JSONContent("scaffold", "default_plugin_json"); data != nil {
				var mfDefaults map[string]any
				if json.Unmarshal(data, &mfDefaults) == nil {
					for k, v := range mfDefaults {
						defaults[k] = v
					}
				}
			}
		}
		pluginData := map[string]any{
			"name":             m.name,
			"slug":             slug,
			"version":          defaults["version"],
			"description":      m.desc,
			"category":         m.category,
			"price_cents":      defaults["price_cents"],
			"billing_interval": m.interval,
			"settings_schema":  defaults["settings_schema"],
		}
		pluginData["platform_tools"] = []any{}
		if m.industry != "" {
			pluginData["industry"] = m.industry
		}
		pj, err := json.MarshalIndent(pluginData, "", "  ")
		if err != nil {
			return generateResultMsg{err: err}
		}
		if err := os.WriteFile(filepath.Join(slug, "plugin.json"), pj, 0o644); err != nil {
			return generateResultMsg{err: err}
		}

		// tools/sample-lookup.json — sample data_store tool
		sampleLookup := map[string]any{
			"name":        "sample_lookup",
			"slug":        "sample-lookup",
			"description": "Look up records from the plugin data store. Call with no args for all, or pass an ID.",
			"parameters": []map[string]any{
				{"name": "id", "type": "string", "description": "Record ID to look up", "required": false},
			},
			"source": map[string]any{
				"type": "data_store",
				"config": map[string]any{
					"key_prefix":          "records:",
					"single_key_template": "records:{{id}}",
				},
			},
		}
		slj, _ := json.MarshalIndent(sampleLookup, "", "  ")
		if err := os.WriteFile(filepath.Join(slug, "tools", "sample-lookup.json"), slj, 0o644); err != nil {
			return generateResultMsg{err: fmt.Errorf("write sample-lookup.json: %w", err)}
		}

		// tools/reference-data.json — sample static tool
		sampleStatic := map[string]any{
			"name":        "reference_data",
			"slug":        "reference-data",
			"description": "Returns reference data definitions. Replace with your own lookup table.",
			"source": map[string]any{
				"type": "static",
				"config": map[string]any{
					"data": []map[string]any{
						{"code": "A", "label": "Category A", "description": "First category"},
						{"code": "B", "label": "Category B", "description": "Second category"},
						{"code": "C", "label": "Category C", "description": "Third category"},
					},
				},
			},
		}
		ssj, _ := json.MarshalIndent(sampleStatic, "", "  ")
		if err := os.WriteFile(filepath.Join(slug, "tools", "reference-data.json"), ssj, 0o644); err != nil {
			return generateResultMsg{err: fmt.Errorf("write reference-data.json: %w", err)}
		}

		// docs/getting-started.md — use manifest template if available
		tmpl := "# {PluginName}\n\n{Description}\n\n## Getting Started\n\n1. Configure settings in the CableKnit dashboard\n2. Add automations in the `automations/` directory\n3. Add skills in the `skills/` directory\n4. Add artifact blueprints in the `blueprints/` directory\n5. Add data source tools in the `tools/` directory\n6. Run `cableknit validate` to check your bundle\n7. Run `cableknit push` to publish\n"
		if mf := api.GetManifest(); mf != nil {
			if t := mf.TextContent("scaffold", "getting_started_template"); t != "" {
				tmpl = t
			}
		}
		md := strings.Replace(strings.Replace(tmpl, "{PluginName}", m.name, 1), "{Description}", m.desc, 1)
		if err := os.WriteFile(filepath.Join(slug, "docs", "getting-started.md"), []byte(md), 0o644); err != nil {
			return generateResultMsg{err: err}
		}

		// images/icon.png — 512x512 placeholder
		icon := image.NewNRGBA(image.Rect(0, 0, 512, 512))
		gray := color.NRGBA{R: 200, G: 200, B: 200, A: 255}
		for y := 0; y < 512; y++ {
			for x := 0; x < 512; x++ {
				icon.SetNRGBA(x, y, gray)
			}
		}
		iconFile, err := os.Create(filepath.Join(slug, "images", "icon.png"))
		if err != nil {
			return generateResultMsg{err: fmt.Errorf("create icon.png: %w", err)}
		}
		if err := png.Encode(iconFile, icon); err != nil {
			iconFile.Close()
			return generateResultMsg{err: fmt.Errorf("encode icon.png: %w", err)}
		}
		iconFile.Close()

		return generateResultMsg{slug: slug}
	}
}
