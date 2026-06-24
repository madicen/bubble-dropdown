package bubbledropdown

import "github.com/charmbracelet/lipgloss"

// Config holds static options for a Dropdown built with New.
type Config struct {
	Options      []string
	InitialIndex int
	Placeholder  string
	MaxVisible   int

	// Styling. Each style has a corresponding Custom* flag so New can tell an
	// intentionally-empty style apart from "not set".
	TriggerStyle lipgloss.Style
	ListStyle    lipgloss.Style
	ItemStyle    lipgloss.Style
	CursorStyle  lipgloss.Style
	AccentColor  string

	CustomTriggerStyle bool
	CustomListStyle    bool
	CustomItemStyle    bool
	CustomCursorStyle  bool
}

// Option configures a Dropdown via New(opts...).
type Option func(*Config)

// WithOptions sets the list of selectable items. An empty slice produces a
// disabled dropdown that shows only the placeholder.
func WithOptions(opts []string) Option {
	return func(c *Config) {
		c.Options = append([]string(nil), opts...)
	}
}

// WithInitialIndex sets which item is shown as selected when the dropdown is
// first rendered. Out-of-range values are clamped to [0, len(options)-1].
func WithInitialIndex(i int) Option {
	return func(c *Config) {
		c.InitialIndex = i
	}
}

// WithPlaceholder sets the text shown when no item is selected or the options
// slice is empty. Defaults to "Select…".
func WithPlaceholder(p string) Option {
	return func(c *Config) {
		c.Placeholder = p
	}
}

// WithMaxVisible sets the maximum number of items shown in the open dropdown
// panel before scrolling kicks in. Defaults to 8.
func WithMaxVisible(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.MaxVisible = n
		}
	}
}

// WithTriggerStyle overrides the lipgloss style applied to the trigger row
// (the closed-state "[  Label ▼ ]" element).
func WithTriggerStyle(s lipgloss.Style) Option {
	return func(c *Config) {
		c.TriggerStyle = s
		c.CustomTriggerStyle = true
	}
}

// WithListStyle overrides the lipgloss style applied to the open dropdown
// panel border. The default is a neutral rounded border (the accent color is
// applied only when set via WithAccentColor).
func WithListStyle(s lipgloss.Style) Option {
	return func(c *Config) {
		c.ListStyle = s
		c.CustomListStyle = true
	}
}

// WithItemStyle overrides the lipgloss style applied to non-highlighted item
// rows in the open panel. The default adds one cell of horizontal padding.
//
// Note: the style should keep symmetric horizontal padding (e.g. Padding(0, 1))
// so item rows align with the panel width computation.
func WithItemStyle(s lipgloss.Style) Option {
	return func(c *Config) {
		c.ItemStyle = s
		c.CustomItemStyle = true
	}
}

// WithCursorStyle overrides the lipgloss style applied to the highlighted
// (hovered / keyboard-cursor) item row. The default is reverse video, or an
// inverted accent color when an accent is set via WithAccentColor.
//
// Note: the style should keep symmetric horizontal padding (e.g. Padding(0, 1))
// so item rows align with the panel width computation.
func WithCursorStyle(s lipgloss.Style) Option {
	return func(c *Config) {
		c.CursorStyle = s
		c.CustomCursorStyle = true
	}
}

// WithAccentColor sets the accent color (any lipgloss color string, e.g. "62"
// or "#7D56F4") used for the panel border, the highlighted item background, and
// the focused trigger arrow. It is opt-in: without it the dropdown renders
// neutrally. Styles set via WithListStyle / WithCursorStyle take precedence
// over the accent color where they overlap.
func WithAccentColor(color string) Option {
	return func(c *Config) {
		c.AccentColor = color
	}
}
