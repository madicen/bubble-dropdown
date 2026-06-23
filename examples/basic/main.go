// Basic example: a small form with three dropdowns — fruit, color, and size.
//
// Demonstrates:
//   - keyboard navigation (Tab to focus, Enter/Space to open, ↑/↓ to navigate)
//   - mouse click to open (via bubblezone zones)
//   - arrow-key cycling without opening (↑/↓ when focused but closed)
//   - smart panel position (opens below, flips above near the bottom)
//
// Run from the bubble-dropdown directory:
//
//	go run ./examples/basic
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
	width    int
	height   int
	zm       *zone.Manager
	dropdowns [3]*bubbledropdown.Dropdown
	focused  int // index of the focused dropdown (for keyboard nav)
	quitting bool
}

func newApp() *appModel {
	zm := zone.New()

	a := &appModel{
		zm:      zm,
		focused: 0,
		dropdowns: [3]*bubbledropdown.Dropdown{
			bubbledropdown.New(
				bubbledropdown.WithOptions([]string{
					"Apple", "Banana", "Cherry", "Date", "Elderberry", "Fig", "Grape",
				}),
				bubbledropdown.WithPlaceholder("Pick a fruit"),
			),
			bubbledropdown.New(
				bubbledropdown.WithOptions([]string{
					"Red", "Green", "Blue", "Yellow", "Purple", "Orange", "Pink",
				}),
				bubbledropdown.WithPlaceholder("Pick a color"),
			),
			bubbledropdown.New(
				bubbledropdown.WithOptions([]string{
					"XS", "S", "M", "L", "XL", "2XL",
				}),
				bubbledropdown.WithPlaceholder("Pick a size"),
				bubbledropdown.WithMaxVisible(6),
			),
		},
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
			// Only Tab when no dropdown is open
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

	// When a panel is open, only that dropdown gets messages.
	openIdx := a.openIndex()
	if openIdx >= 0 {
		a.dropdowns[openIdx], cmd = a.dropdowns[openIdx].Update(msg)
		return a, cmd
	}

	// Route mouse press to the correct dropdown via zone hit-test.
	if m, ok := msg.(tea.MouseMsg); ok &&
		m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
		for i, dd := range a.dropdowns {
			z := a.zm.Get(dropdownZoneID(i))
			if z != nil && z.InBounds(m) {
				// Move focus to this dropdown on click
				a.dropdowns[a.focused].SetFocused(false)
				a.focused = i
				dd.SetFocused(true)
				a.dropdowns[i], cmd = a.dropdowns[i].Update(msg)
				return a, cmd
			}
		}
		return a, nil
	}

	// Broadcast key events to the focused dropdown; broadcast window resize to all.
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

func dropdownZoneID(i int) string { return "dropdown-" + strconv.Itoa(i) }

// ── View ─────────────────────────────────────────────────────────────────────

func (a *appModel) View() string {
	if a.width <= 0 {
		a.width = 60
	}
	if a.height <= 0 {
		a.height = 20
	}

	mainView := a.buildMainView()

	// Composite open panels (only one can be open at a time, but the loop is safe).
	for _, dd := range a.dropdowns {
		mainView = dd.ViewWithOverlay(mainView, a.width, a.height)
	}

	return a.zm.Scan(mainView)
}

func (a *appModel) buildMainView() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	labelStyle := lipgloss.NewStyle().Width(8)

	// ── Row layout ──────────────────────────────────────────────────────────
	// Place each dropdown at a fixed column so SetBounds matches the zone.
	const labelW = 8  // "Fruit:  " etc.
	const startRow = 3

	tw0, th0 := a.dropdowns[0].TriggerSize()
	tw1, th1 := a.dropdowns[1].TriggerSize()
	tw2, th2 := a.dropdowns[2].TriggerSize()

	a.dropdowns[0].SetBounds(startRow, labelW, tw0, th0)
	a.dropdowns[1].SetBounds(startRow+2, labelW, tw1, th1)
	a.dropdowns[2].SetBounds(startRow+4, labelW, tw2, th2)

	row0 := labelStyle.Render("Fruit:") + a.zm.Mark(dropdownZoneID(0), a.dropdowns[0].TriggerView())
	row1 := labelStyle.Render("Color:") + a.zm.Mark(dropdownZoneID(1), a.dropdowns[1].TriggerView())
	row2 := labelStyle.Render("Size:")  + a.zm.Mark(dropdownZoneID(2), a.dropdowns[2].TriggerView())

	// ── Summary ──────────────────────────────────────────────────────────────
	summary := fmt.Sprintf(
		"%s  %s  %s",
		a.dropdowns[0].Selected(),
		a.dropdowns[1].Selected(),
		a.dropdowns[2].Selected(),
	)

	help := dimStyle.Render("tab: next  ↑↓: cycle  enter/space: open  q: quit")

	lines := []string{
		titleStyle.Render("bubble-dropdown example"),
		"",
		"",
		row0,
		"",
		row1,
		"",
		row2,
		"",
		dimStyle.Render("Selection: ") + summary,
		"",
		help,
	}
	return strings.Join(lines, "\n")
}
