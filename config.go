package bubbledropdown

import "github.com/charmbracelet/lipgloss"

// Config holds static options for a Dropdown built with New.
type Config struct {
	Options      []string
	InitialIndex int
	Placeholder  string
	MaxVisible   int
	TriggerStyle lipgloss.Style
	ListStyle    lipgloss.Style
	CustomTriggerStyle bool
	CustomListStyle    bool
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
// panel border. The default is a rounded border in the accent color.
func WithListStyle(s lipgloss.Style) Option {
	return func(c *Config) {
		c.ListStyle = s
		c.CustomListStyle = true
	}
}
