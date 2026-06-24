package bubbledropdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces a deterministic color profile so rendered views contain
// predictable ANSI sequences regardless of whether the test runs under a TTY.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	m.Run()
}

// fgSeq / bgSeq return the SGR parameter substring lipgloss emits for the given
// lipgloss color string under the TrueColor profile (e.g. "38;2;255;0;0").
func fgSeq(t *testing.T, color string) string {
	t.Helper()
	return termenv.TrueColor.Color(color).Sequence(false)
}

func bgSeq(t *testing.T, color string) string {
	t.Helper()
	return termenv.TrueColor.Color(color).Sequence(true)
}

func TestSetAccentColorNilSafe(t *testing.T) {
	var d *Dropdown
	// Must not panic.
	d.SetAccentColor("#FF0000")
	if got := d.AccentColor(); got != "" {
		t.Fatalf("nil AccentColor() = %q, want empty string", got)
	}
}

func TestSetAccentColorStoresValue(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}))
	d.SetAccentColor("#FF0000")
	if got := d.AccentColor(); got != "#FF0000" {
		t.Fatalf("AccentColor() = %q, want %q", got, "#FF0000")
	}
}

func TestSetAccentColorEmptyResetsToDefault(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}), WithAccentColor("#FF0000"))
	d.SetAccentColor("")
	if got := d.AccentColor(); got != accentColor {
		t.Fatalf("AccentColor() = %q, want default %q", got, accentColor)
	}
}

func TestSetAccentColorTriggerArrow(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}))
	d.SetFocused(true)
	d.SetAccentColor("#FF0000")

	view := d.TriggerView()
	want := fgSeq(t, "#FF0000")
	if !strings.Contains(view, want) {
		t.Fatalf("TriggerView() = %q, want it to contain accent fg sequence %q", view, want)
	}
}

func TestSetAccentColorRecolorsOpenPanel(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}))
	d, _ = d.doOpen()
	if !d.Open() {
		t.Fatal("dropdown should be open after doOpen()")
	}

	d.SetAccentColor("#FF0000")

	panel := d.list.View()
	wantBorder := fgSeq(t, "#FF0000")
	wantCursor := bgSeq(t, "#FF0000")
	if !strings.Contains(panel, wantBorder) {
		t.Fatalf("panel View() missing accent border fg sequence %q\n%q", wantBorder, panel)
	}
	if !strings.Contains(panel, wantCursor) {
		t.Fatalf("panel View() missing accent cursor bg sequence %q\n%q", wantCursor, panel)
	}

	// The composited overlay view should also carry the new accent.
	overlayView := d.ViewWithOverlay("main\nview", 40, 20)
	if !strings.Contains(overlayView, wantBorder) {
		t.Fatalf("ViewWithOverlay() missing accent border fg sequence %q", wantBorder)
	}
}

func TestSetAccentColorCustomStylePrecedence(t *testing.T) {
	customBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF00"))

	d := New(
		WithOptions([]string{"a", "b", "c"}),
		WithListStyle(customBorder),
	)
	d, _ = d.doOpen()
	d.SetAccentColor("#FF0000")

	panel := d.list.View()

	// Custom border color is preserved (does NOT switch to the new accent).
	keptBorder := fgSeq(t, "#00FF00")
	if !strings.Contains(panel, keptBorder) {
		t.Fatalf("custom border color %q should be preserved\n%q", keptBorder, panel)
	}
	// The cursor row falls back to the accent and DOES change.
	wantCursor := bgSeq(t, "#FF0000")
	if !strings.Contains(panel, wantCursor) {
		t.Fatalf("cursor row should use new accent bg %q\n%q", wantCursor, panel)
	}
}
