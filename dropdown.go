// Package bubbledropdown provides Dropdown: a trigger element that opens a
// scrollable selection panel as a bubble-overlay modal. Embed it in your
// BubbleTea model, call SetBounds and forward messages, then call
// ViewWithOverlay to composite the open panel over your main view.
//
// Usage overview:
//
//	d := bubbledropdown.New(
//	    bubbledropdown.WithOptions([]string{"Apple", "Banana", "Cherry"}),
//	    bubbledropdown.WithPlaceholder("Pick a fruit"),
//	)
//	d.SetZoneManager(zm) // optional; enables bubblezone hit-testing
//
// In your View:
//
//	tw, th := d.TriggerSize()
//	d.SetBounds(row, col, tw, th)
//	mainView := zm.Mark("my-dropdown", d.TriggerView())
//	return d.ViewWithOverlay(mainView, width, height)
//
// In your Update, forward all messages:
//
//	d, cmd = d.Update(msg)
//
// On ItemChosenMsg, call d.SetSelectedIndex(msg.Index).
package bubbledropdown

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	overlay "github.com/madicen/bubble-overlay"
)

// dropdownArrow is the glyph appended to the trigger label.
const dropdownArrow = "▼"

// Dropdown is a trigger + panel component. Use New to create one; the zero
// value is not valid. All methods are safe to call on a nil pointer (they
// are no-ops / return zero values) so partial initialisation is safe.
type Dropdown struct {
	options     []string
	selectedIdx int
	placeholder string
	maxVisible  int

	// trigger position and size (set by SetBounds each frame)
	row, col int
	w, h     int

	list listModel
	open bool

	// focused controls the trigger's highlighted-arrow appearance for
	// keyboard-only navigation. The host toggles this via SetFocused.
	focused bool

	// ignoreNextRelease: set to true when the dropdown opens on a press so the
	// same-click release does not immediately confirm the first highlighted item.
	ignoreNextRelease bool

	// zoneManager: when set, bubblezone hit-testing is used for the trigger.
	// The host must call zm.Scan on the final view string.
	zoneManager *zone.Manager

	// overlay geometry cached by ViewWithOverlay; used by Update for mouse offset.
	lastOverlayLeft   int
	lastOverlayTop    int
	lastPanelW        int
	lastPanelH        int
	lastViewWidth     int
	lastViewHeight    int

	// openAbove: true when the panel was placed above the trigger (smart position).
	openAbove bool

	// styles
	triggerStyle       lipgloss.Style
	customTriggerStyle bool
}

// New creates a Dropdown configured with the given options.
func New(opts ...Option) *Dropdown {
	cfg := &Config{
		Placeholder: "Select…",
		MaxVisible:  defaultMaxVisible,
	}
	for _, o := range opts {
		o(cfg)
	}

	d := &Dropdown{
		options:     cfg.Options,
		placeholder: cfg.Placeholder,
		maxVisible:  cfg.MaxVisible,
	}

	if cfg.CustomTriggerStyle {
		d.triggerStyle = cfg.TriggerStyle
		d.customTriggerStyle = true
	}

	// Clamp initial index
	if len(cfg.Options) > 0 {
		idx := cfg.InitialIndex
		if idx < 0 {
			idx = 0
		}
		if idx >= len(cfg.Options) {
			idx = len(cfg.Options) - 1
		}
		d.selectedIdx = idx
	}

	return d
}

// ── Accessors ───────────────────────────────────────────────────────────────

// Open returns true when the dropdown panel is currently displayed.
func (d *Dropdown) Open() bool {
	if d == nil {
		return false
	}
	return d.open
}

// Focused returns true when the trigger is marked as focused (highlighted arrow).
func (d *Dropdown) Focused() bool {
	if d == nil {
		return false
	}
	return d.focused
}

// SetFocused marks the trigger as focused or unfocused. When focused, the ▼
// arrow is rendered in the accent color so keyboard-only users have a visible
// indicator.
func (d *Dropdown) SetFocused(f bool) {
	if d == nil {
		return
	}
	d.focused = f
}

// Selected returns the currently selected option string, or the placeholder
// if nothing is selected.
func (d *Dropdown) Selected() string {
	if d == nil || len(d.options) == 0 {
		if d != nil {
			return d.placeholder
		}
		return ""
	}
	return d.options[d.selectedIdx]
}

// SelectedIndex returns the zero-based index of the selected option.
func (d *Dropdown) SelectedIndex() int {
	if d == nil {
		return 0
	}
	return d.selectedIdx
}

// SetSelectedIndex updates which item appears selected. Out-of-range values
// are clamped. This is typically called after receiving ItemChosenMsg.
func (d *Dropdown) SetSelectedIndex(i int) {
	if d == nil || len(d.options) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= len(d.options) {
		i = len(d.options) - 1
	}
	d.selectedIdx = i
}

// SetZoneManager sets the bubblezone manager. When set, the host marks the
// trigger with zm.Mark and the dropdown skips its own coordinate hit-test
// (trusting the zone was already checked). The host must call zm.Scan on the
// final view string.
func (d *Dropdown) SetZoneManager(zm *zone.Manager) {
	if d == nil {
		return
	}
	d.zoneManager = zm
}

// ── Bounds ──────────────────────────────────────────────────────────────────

// SetBounds records where the trigger is drawn (0-based row, col) and its
// display size (w cells wide, h rows tall). Call this every frame before
// ViewWithOverlay so the panel is positioned correctly.
//
// If w or h is zero, TriggerSize() is used for the missing dimension.
func (d *Dropdown) SetBounds(row, col, w, h int) {
	if d == nil {
		return
	}
	d.row = row
	d.col = col
	tw, th := d.TriggerSize()
	if w <= 0 {
		w = tw
	}
	if h <= 0 {
		h = th
	}
	d.w = w
	d.h = h
}

// ── Trigger view ─────────────────────────────────────────────────────────────

// selectedLabel returns the label to show in the trigger.
func (d *Dropdown) selectedLabel() string {
	if d == nil || len(d.options) == 0 {
		if d != nil {
			return d.placeholder
		}
		return ""
	}
	return d.options[d.selectedIdx]
}

// maxLabelWidth returns the display-cell width of the longest option (or
// placeholder), used to keep the trigger at a stable width across selections.
func (d *Dropdown) maxLabelWidth() int {
	w := lipgloss.Width(d.placeholder)
	for _, opt := range d.options {
		if ow := lipgloss.Width(opt); ow > w {
			w = ow
		}
	}
	return w
}

// TriggerSize returns the stable display-cell width and height (always 1) of
// the trigger. Width is computed from the longest option so the trigger does
// not resize as the selection changes.
//
// Use this when building your layout and when calling SetBounds.
func (d *Dropdown) TriggerSize() (width, height int) {
	if d == nil {
		return 8, 1
	}
	// "[ " + label + " " + arrow + " ]" = 6 + maxLabelWidth
	return d.maxLabelWidth() + 6, 1
}

// TriggerView renders the closed-state trigger: "[ Label ▼ ]".
// When focused, the ▼ arrow is rendered in the accent color.
func (d *Dropdown) TriggerView() string {
	if d == nil {
		return "[ Select… ▼ ]"
	}
	tw, _ := d.TriggerSize()
	labelW := tw - 6 // subtract "[ " + " ▼ ]"
	label := lipgloss.NewStyle().Width(labelW).Render(d.selectedLabel())

	arrow := dropdownArrow
	if d.focused {
		arrow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(accentColor)).
			Bold(true).
			Render(arrow)
	}

	trigger := "[ " + label + " " + arrow + " ]"

	if d.customTriggerStyle {
		return d.triggerStyle.Render(trigger)
	}
	return trigger
}

// ── Overlay ──────────────────────────────────────────────────────────────────

// ViewWithOverlay returns the view to display. When the dropdown is open it
// composites the panel over mainView using bubble-overlay. Call this at the
// end of your View method; it also caches overlay geometry for Update.
//
//	mainView := buildMain()
//	mainView = d.ViewWithOverlay(mainView, width, height)
//	return zm.Scan(mainView)
func (d *Dropdown) ViewWithOverlay(mainView string, viewWidth, viewHeight int) string {
	if d == nil || !d.open {
		return mainView
	}

	panelContent := d.list.View()
	panelW, panelH := overlay.ModalCellSize(panelContent)

	topPad, leftPad := d.computeOrigin(panelW, panelH, viewWidth, viewHeight)

	d.lastOverlayLeft = leftPad
	d.lastOverlayTop = topPad
	d.lastPanelW = panelW
	d.lastPanelH = panelH
	d.lastViewWidth = viewWidth
	d.lastViewHeight = viewHeight

	return overlay.OverlayView(mainView, panelContent, viewWidth, viewHeight, topPad, leftPad)
}

// computeOrigin decides whether to open the panel below or above the trigger,
// then clamps to the viewport.
func (d *Dropdown) computeOrigin(panelW, panelH, viewW, viewH int) (top, left int) {
	belowTop := d.row + d.h
	aboveTop := d.row - panelH

	// Smart position: prefer below; flip above if there's not enough room.
	if belowTop+panelH > viewH && aboveTop >= 0 {
		d.openAbove = true
		top = aboveTop
	} else {
		d.openAbove = false
		top = belowTop
	}
	left = d.col

	return overlay.Fixed(top, left).ClampedOrigin(panelW, panelH, viewW, viewH)
}

// ── Update ───────────────────────────────────────────────────────────────────

// Update handles all BubbleTea messages. Forward every tea.Msg here.
//
//   - When the dropdown is open it routes keys and mouse events to the list panel.
//   - On ItemChosenMsg / ItemCanceledMsg it closes the dropdown; the host should
//     also handle ItemChosenMsg to call SetSelectedIndex(msg.Index).
//   - When closed, a mouse press on the trigger (or zone hit) opens the panel.
func (d *Dropdown) Update(msg tea.Msg) (*Dropdown, tea.Cmd) {
	if d == nil {
		return d, nil
	}

	switch m := msg.(type) {

	// ── Window resize ──────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		d.lastViewWidth = m.Width
		d.lastViewHeight = m.Height
		if d.open {
			// Recompute overlay origin so mouse coords remain correct.
			top, left := d.computeOrigin(d.lastPanelW, d.lastPanelH, m.Width, m.Height)
			d.lastOverlayLeft = left
			d.lastOverlayTop = top
		}
		return d, nil

	// ── Key events ─────────────────────────────────────────────────────────
	case tea.KeyMsg:
		if d.open {
			// Delegate all keys to the list (up/down/enter/esc).
			list, cmd := d.list.Update(m)
			d.list = list
			return d, cmd
		}
		// When focused but closed: Enter/Space opens; ↑/↓ cycles without opening.
		if d.focused && len(d.options) > 0 {
			switch m.String() {
			case "enter", " ":
				return d.doOpen()
			case "up", "k":
				if d.selectedIdx > 0 {
					d.selectedIdx--
				}
				return d, nil
			case "down", "j":
				if d.selectedIdx < len(d.options)-1 {
					d.selectedIdx++
				}
				return d, nil
			}
		}
		return d, nil

	// ── Mouse events ────────────────────────────────────────────────────────
	case tea.MouseMsg:
		if d.open {
			// Ignore the first left-button release after we opened (same click that
			// opened the panel), so it does not immediately confirm an option.
			if d.ignoreNextRelease &&
				m.Action == tea.MouseActionRelease &&
				m.Button == tea.MouseButtonLeft {
				next := *d
				next.ignoreNextRelease = false
				return &next, nil
			}

			inPanel := overlay.CellInModal(
				m.X, m.Y,
				d.lastOverlayTop, d.lastOverlayLeft,
				d.lastPanelW, d.lastPanelH,
			)

			if !inPanel {
				// Click outside closes the panel.
				if m.Action == tea.MouseActionPress {
					next := *d
					next.open = false
					return &next, func() tea.Msg { return ItemCanceledMsg{} }
				}
				// Pass non-press out-of-panel events through (e.g. motion).
				return d, nil
			}

			// Translate to panel-relative coordinates before forwarding.
			relMsg := tea.MouseMsg{
				X:      m.X - d.lastOverlayLeft,
				Y:      m.Y - d.lastOverlayTop,
				Button: m.Button,
				Action: m.Action,
				Alt:    m.Alt,
				Ctrl:   m.Ctrl,
				Shift:  m.Shift,
			}
			list, cmd := d.list.Update(relMsg)
			d.list = list
			return d, cmd
		}

		// Closed state: detect a click on the trigger.
		if m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonLeft {
			inBounds := d.zoneManager != nil ||
				(m.X >= d.col && m.X < d.col+d.w &&
					m.Y >= d.row && m.Y < d.row+d.h)
			if inBounds {
				return d.doOpen()
			}
		}
		return d, nil

	// ── Result messages ─────────────────────────────────────────────────────
	case ItemChosenMsg:
		if !d.open {
			return d, nil
		}
		next := *d
		next.selectedIdx = m.Index
		next.open = false
		return &next, nil

	case ItemCanceledMsg:
		if !d.open {
			return d, nil
		}
		next := *d
		next.open = false
		return &next, nil
	}

	// Forward unknown messages to the list when open.
	if d.open {
		list, cmd := d.list.Update(msg)
		d.list = list
		return d, cmd
	}
	return d, nil
}

// doOpen creates a fresh listModel and opens the panel.
func (d *Dropdown) doOpen() (*Dropdown, tea.Cmd) {
	next := *d

	// Minimum content width: trigger width minus 2 border cells.
	minW := d.w - 2
	if minW < 1 {
		minW = 1
	}

	next.list = newListModel(d.options, d.selectedIdx, d.maxVisible, minW)
	next.open = true
	next.ignoreNextRelease = true

	// Pre-compute overlay origin so the first mouse event uses correct offsets.
	panelW, panelH := next.list.PanelSize()
	top, left := next.computeOrigin(panelW, panelH, next.lastViewWidth, next.lastViewHeight)
	next.lastOverlayLeft = left
	next.lastOverlayTop = top
	next.lastPanelW = panelW
	next.lastPanelH = panelH

	return &next, nil
}
