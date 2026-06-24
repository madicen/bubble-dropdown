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

	listStyle       lipgloss.Style
	customListStyle bool

	itemStyle       lipgloss.Style
	customItemStyle bool

	cursorStyle       lipgloss.Style
	customCursorStyle bool

	// accent is the color used for the panel border, the highlighted item, and
	// the focused trigger arrow. Empty means neutral (uncolored): a plain
	// border, a reverse-video highlight, and a bold-only focused arrow.
	accent string
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

	// accent stays empty (neutral) unless the caller opts in.
	if cfg.AccentColor != "" {
		d.accent = cfg.AccentColor
	}
	if cfg.CustomTriggerStyle {
		d.triggerStyle = cfg.TriggerStyle
		d.customTriggerStyle = true
	}
	if cfg.CustomListStyle {
		d.listStyle = cfg.ListStyle
		d.customListStyle = true
	}
	if cfg.CustomItemStyle {
		d.itemStyle = cfg.ItemStyle
		d.customItemStyle = true
	}
	if cfg.CustomCursorStyle {
		d.cursorStyle = cfg.CursorStyle
		d.customCursorStyle = true
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
// arrow is emphasized (bold, plus the accent color when one is set) so
// keyboard-only users have a visible indicator.
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

// AccentColor returns the current accent color used for the panel border, the
// highlighted item, and the focused trigger arrow. An empty string means the
// dropdown is neutral (uncolored). Lets consumers detect changes without
// shadow-tracking the value themselves.
func (d *Dropdown) AccentColor() string {
	if d == nil {
		return ""
	}
	return d.accent
}

// SetAccentColor changes the accent color at runtime. Pass any lipgloss color
// string (e.g. "62" or "#7D56F4"); an empty string clears the accent and
// returns the dropdown to its neutral, uncolored appearance. This is the
// live-theming counterpart to WithAccentColor: consumers with a user-editable
// theme can recolor the dropdown in place instead of rebuilding it.
//
// The focused trigger arrow re-reads the accent on each render, so it updates
// automatically. If the panel is currently open it is recolored immediately so
// the change is visible without waiting for the next open. Custom styles set
// via WithListStyle / WithItemStyle / WithCursorStyle still take precedence.
func (d *Dropdown) SetAccentColor(color string) {
	if d == nil {
		return
	}
	d.accent = color
	if d.open {
		d.list.setStyles(d.panelStyles())
	}
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
//
// Focus is indicated on the ▼ arrow. For the default (unstyled) trigger the
// arrow is drawn in the accent color so keyboard-only users get a visible
// affordance. When a custom trigger style is set, the arrow instead keeps the
// trigger's own color (emphasized with bold) so the glyph always matches the
// caller's styling rather than clashing with the accent.
func (d *Dropdown) TriggerView() string {
	if d == nil {
		return "[ Select… ▼ ]"
	}
	tw, _ := d.TriggerSize()
	labelW := tw - 6 // subtract "[ " + " ▼ ]"
	label := lipgloss.NewStyle().Width(labelW).Render(d.selectedLabel())

	// Custom trigger style: render the element with the caller's style so every
	// glyph (including the arrow) shares the same color. Pre-styling the arrow
	// separately would embed its own color and reset sequence, leaving the
	// arrow — and the text after it — out of sync with the rest of the trigger.
	if d.customTriggerStyle {
		if d.focused {
			// Emphasize focus with bold on the arrow while keeping its color in
			// sync. Render in segments so the bold-only arrow does not require a
			// nested reset that would break the surrounding style.
			pre := d.triggerStyle.Render("[ " + label + " ")
			arrow := d.triggerStyle.Bold(true).Render(dropdownArrow)
			post := d.triggerStyle.Render(" ]")
			return pre + arrow + post
		}
		return d.triggerStyle.Render("[ " + label + " " + dropdownArrow + " ]")
	}

	arrow := dropdownArrow
	if d.focused {
		// Emphasize focus with bold. When an accent is set, also color the
		// arrow; otherwise it stays neutral (bold only).
		style := lipgloss.NewStyle().Bold(true)
		if d.accent != "" {
			style = style.Foreground(lipgloss.Color(d.accent))
		}
		arrow = style.Render(arrow)
	}

	return "[ " + label + " " + arrow + " ]"
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

// panelStyles resolves the styles for the open panel: it starts from the
// accent-derived defaults and overlays any caller-provided custom styles.
func (d *Dropdown) panelStyles() listStyles {
	styles := defaultListStyles(d.accent)
	if d.customListStyle {
		styles.border = d.listStyle
	}
	if d.customItemStyle {
		styles.normal = d.itemStyle
	}
	if d.customCursorStyle {
		styles.cursor = d.cursorStyle
	}
	return styles
}

// doOpen creates a fresh listModel and opens the panel.
func (d *Dropdown) doOpen() (*Dropdown, tea.Cmd) {
	next := *d

	// Minimum content width: trigger width minus 2 border cells.
	minW := d.w - 2
	if minW < 1 {
		minW = 1
	}

	next.list = newListModel(d.options, d.selectedIdx, d.maxVisible, minW, d.panelStyles())
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
