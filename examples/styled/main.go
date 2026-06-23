// Styled example: two dropdowns with fully customized appearance.
//
// Demonstrates every styling option:
//   - WithAccentColor  — recolors the border, highlight, and focused arrow
//   - WithTriggerStyle — restyles the closed "[ Label ▼ ]" element
//   - WithListStyle    — restyles the open panel's border
//   - WithItemStyle    — restyles non-highlighted item rows
//   - WithCursorStyle  — restyles the highlighted (hovered/cursor) row
//
// Run from the bubble-dropdown directory:
//
//	go run ./examples/styled
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	bubbledropdown "github.com/madicen/bubble-dropdown"
	"github.com/muesli/termenv"
)

func main() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	p := tea.NewProgram(newApp(), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ── App model ────────────────────────────────────────────────────────────────

type appModel struct {
	width     int
	height    int
	zm        *zone.Manager
	dropdowns [2]*bubbledropdown.Dropdown
	focused   int
	quitting  bool
}

func newApp() *appModel {
	zm := zone.New()

	// Theme 1: "Violet" — recolor everything with a single accent color, plus a
	// bold, padded trigger. The panel border, highlight, and focused arrow all
	// follow the accent automatically.
	violet := "#7D56F4"
	fruit := bubbledropdown.New(
		bubbledropdown.WithOptions([]string{
			"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape",
		}),
		bubbledropdown.WithPlaceholder("Pick a fruit"),
		bubbledropdown.WithAccentColor(violet),
		bubbledropdown.WithTriggerStyle(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(violet)),
		),
	)

	// Theme 2: "Terminal green" — fully custom panel styles. A thick border, a
	// dim normal row, and an inverse-green cursor row. Keep symmetric horizontal
	// padding on item/cursor styles so rows align with the panel width.
	green := "#2ECC71"
	color := bubbledropdown.New(
		bubbledropdown.WithOptions([]string{
			"Red", "Green", "Blue", "Yellow", "Purple", "Orange", "Pink",
		}),
		bubbledropdown.WithPlaceholder("Pick a color"),
		bubbledropdown.WithTriggerStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color(green)).
				Background(lipgloss.Color("236")).
				Bold(true),
		),
		bubbledropdown.WithListStyle(
			lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color(green)),
		),
		bubbledropdown.WithItemStyle(
			lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(lipgloss.Color("245")),
		),
		bubbledropdown.WithCursorStyle(
			lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true).
				Background(lipgloss.Color(green)).
				Foreground(lipgloss.Color("16")),
		),
	)

	a := &appModel{
		zm:        zm,
		focused:   0,
		dropdowns: [2]*bubbledropdown.Dropdown{fruit, color},
	}

	for _, dd := range a.dropdowns {
		dd.SetZoneManager(zm)
	}
	a.dropdowns[0].SetFocused(true)

	return a
}

func (a *appModel) Init() tea.Cmd { return nil }

// ── Update ───────────────────────────────────────────────────────────────────

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			a.quitting = true
			return a, tea.Quit
		case "tab", "shift+tab":
			if !a.anyOpen() {
				prev := a.focused
				if msg.String() == "tab" {
					a.focused = (a.focused + 1) % len(a.dropdowns)
				} else {
					a.focused = (a.focused + len(a.dropdowns) - 1) % len(a.dropdowns)
				}
				a.dropdowns[prev].SetFocused(false)
				a.dropdowns[a.focused].SetFocused(true)
				return a, nil
			}
		}
	}

	var cmd tea.Cmd

	if openIdx := a.openIndex(); openIdx >= 0 {
		a.dropdowns[openIdx], cmd = a.dropdowns[openIdx].Update(msg)
		return a, cmd
	}

	if m, ok := msg.(tea.MouseMsg); ok &&
		m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
		for i, dd := range a.dropdowns {
			z := a.zm.Get(dropdownZoneID(i))
			if z != nil && z.InBounds(m) {
				a.dropdowns[a.focused].SetFocused(false)
				a.focused = i
				dd.SetFocused(true)
				a.dropdowns[i], cmd = a.dropdowns[i].Update(msg)
				return a, cmd
			}
		}
		return a, nil
	}

	switch msg.(type) {
	case tea.KeyMsg:
		a.dropdowns[a.focused], cmd = a.dropdowns[a.focused].Update(msg)
	default:
		for i := range a.dropdowns {
			var c tea.Cmd
			a.dropdowns[i], c = a.dropdowns[i].Update(msg)
			if c != nil {
				cmd = c
			}
		}
	}
	return a, cmd
}

func (a *appModel) anyOpen() bool {
	for _, dd := range a.dropdowns {
		if dd.Open() {
			return true
		}
	}
	return false
}

func (a *appModel) openIndex() int {
	for i, dd := range a.dropdowns {
		if dd.Open() {
			return i
		}
	}
	return -1
}

func dropdownZoneID(i int) string { return "styled-dropdown-" + strconv.Itoa(i) }

// ── View ─────────────────────────────────────────────────────────────────────

func (a *appModel) View() string {
	if a.width <= 0 {
		a.width = 60
	}
	if a.height <= 0 {
		a.height = 20
	}

	mainView := a.buildMainView()
	for _, dd := range a.dropdowns {
		mainView = dd.ViewWithOverlay(mainView, a.width, a.height)
	}
	return a.zm.Scan(mainView)
}

func (a *appModel) buildMainView() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	labelStyle := lipgloss.NewStyle().Width(8)

	const labelW = 8
	const startRow = 3

	tw0, th0 := a.dropdowns[0].TriggerSize()
	tw1, th1 := a.dropdowns[1].TriggerSize()

	a.dropdowns[0].SetBounds(startRow, labelW, tw0, th0)
	a.dropdowns[1].SetBounds(startRow+2, labelW, tw1, th1)

	row0 := labelStyle.Render("Fruit:") + a.zm.Mark(dropdownZoneID(0), a.dropdowns[0].TriggerView())
	row1 := labelStyle.Render("Color:") + a.zm.Mark(dropdownZoneID(1), a.dropdowns[1].TriggerView())

	summary := fmt.Sprintf("%s  %s",
		a.dropdowns[0].Selected(),
		a.dropdowns[1].Selected(),
	)

	help := dimStyle.Render("tab: next  ↑↓: cycle  enter/space: open  q: quit")

	lines := []string{
		titleStyle.Render("bubble-dropdown — styled example"),
		"",
		"",
		row0,
		"",
		row1,
		"",
		dimStyle.Render("Selection: ") + summary,
		"",
		help,
	}
	return strings.Join(lines, "\n")
}
