// Package bubbledropdown — internal list panel rendered inside the overlay.

package bubbledropdown

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const defaultMaxVisible = 8

// accentColor is the default highlight / border color.
const accentColor = "62"

// listStyles bundles the lipgloss styles used to render the open panel.
type listStyles struct {
	border lipgloss.Style
	normal lipgloss.Style
	cursor lipgloss.Style
}

// defaultListStyles returns the built-in panel styles for the given accent
// color: a rounded border, plain padded rows, and an inverted-accent cursor.
func defaultListStyles(accent string) listStyles {
	if accent == "" {
		accent = accentColor
	}
	return listStyles{
		border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(accent)),
		normal: lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1),
		cursor: lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).
			Background(lipgloss.Color(accent)).
			Foreground(lipgloss.Color("255")),
	}
}

// listModel is the open dropdown panel. It is created fresh each time the
// dropdown opens; its View is composited by Dropdown.ViewWithOverlay.
//
// Mouse coordinates received by Update are expected to be relative to the
// panel's top-left cell (0,0 = top-left border rune). Dropdown.Update handles
// the translation from screen coords before forwarding here.
type listModel struct {
	items      []string
	cursor     int // absolute index of the highlighted item
	offset     int // index of the first visible item (scroll offset)
	maxVisible int
	minContentW int // minimum content-area width; expands to fit longest item

	borderStyle lipgloss.Style
	normalStyle lipgloss.Style
	cursorStyle lipgloss.Style
}

func newListModel(items []string, initialCursor, maxVisible, minContentW int, styles listStyles) listModel {
	if maxVisible <= 0 {
		maxVisible = defaultMaxVisible
	}
	if initialCursor < 0 {
		initialCursor = 0
	}
	if len(items) > 0 && initialCursor >= len(items) {
		initialCursor = len(items) - 1
	}

	m := listModel{
		items:       items,
		cursor:      initialCursor,
		maxVisible:  maxVisible,
		minContentW: minContentW,
		borderStyle: styles.border,
		normalStyle: styles.normal,
		cursorStyle: styles.cursor,
	}
	m.scrollToCursor()
	return m
}

// setStyles replaces the panel's border, normal, and cursor styles in place.
// Dropdown.SetAccentColor uses this to recolor an already-open panel without
// rebuilding the listModel (which would reset cursor/scroll state).
func (m *listModel) setStyles(s listStyles) {
	if m == nil {
		return
	}
	m.borderStyle = s.border
	m.normalStyle = s.normal
	m.cursorStyle = s.cursor
}

// contentWidth returns the rendered width of each item row (excluding border cells).
func (m listModel) contentWidth() int {
	w := m.minContentW
	// +2 for the 1-cell padding on each side
	for _, item := range m.items {
		if iw := lipgloss.Width(item) + 2; iw > w {
			w = iw
		}
	}
	if w < 4 {
		w = 4
	}
	return w
}

// PanelSize returns the display-cell dimensions of View().
// Use this before opening so Dropdown can compute the overlay origin.
func (m listModel) PanelSize() (w, h int) {
	visible := len(m.items)
	if visible > m.maxVisible {
		visible = m.maxVisible
	}
	// +2 for top and bottom border
	return m.contentWidth() + 2, visible + 2
}

// View renders the bordered item list.
func (m listModel) View() string {
	cw := m.contentWidth()

	normal := m.normalStyle.Width(cw)
	cursor := m.cursorStyle.Width(cw)

	visibleEnd := m.offset + m.maxVisible
	if visibleEnd > len(m.items) {
		visibleEnd = len(m.items)
	}

	rows := make([]string, 0, visibleEnd-m.offset)
	for i := m.offset; i < visibleEnd; i++ {
		if i == m.cursor {
			rows = append(rows, cursor.Render(m.items[i]))
		} else {
			rows = append(rows, normal.Render(m.items[i]))
		}
	}

	return m.borderStyle.Render(strings.Join(rows, "\n"))
}

func (m listModel) scrollToCursor() listModel {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.maxVisible {
		m.offset = m.cursor - m.maxVisible + 1
	}
	return m
}

// Update handles key and mouse events for the open panel.
// Mouse coords must be relative to the panel's top-left (see Dropdown.Update).
func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m = m.scrollToCursor()
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m = m.scrollToCursor()
			}
			return m, nil
		case "enter", " ":
			if len(m.items) == 0 {
				return m, nil
			}
			idx := m.cursor
			val := m.items[idx]
			return m, func() tea.Msg { return ItemChosenMsg{Index: idx, Value: val} }
		case "esc":
			return m, func() tea.Msg { return ItemCanceledMsg{} }
		}

	case tea.MouseMsg:
		// Relative coords: row 0 = top border, row 1 = first item, etc.
		switch msg.Action {
		case tea.MouseActionMotion:
			itemRow := msg.Y - 1 // 0-based item index relative to offset
			if itemRow >= 0 {
				abs := m.offset + itemRow
				if abs >= 0 && abs < len(m.items) {
					m.cursor = abs
				}
			}
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				itemRow := msg.Y - 1
				if itemRow >= 0 {
					abs := m.offset + itemRow
					if abs >= 0 && abs < len(m.items) {
						m.cursor = abs
						val := m.items[abs]
						return m, func() tea.Msg { return ItemChosenMsg{Index: abs, Value: val} }
					}
				}
			}
		case tea.MouseActionRelease:
			// handled via press
		}

		// Scroll wheel
		if msg.Button == tea.MouseButtonWheelUp {
			if m.cursor > 0 {
				m.cursor--
				m = m.scrollToCursor()
			}
		}
		if msg.Button == tea.MouseButtonWheelDown {
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m = m.scrollToCursor()
			}
		}
	}
	return m, nil
}
