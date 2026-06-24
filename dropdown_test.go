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

func TestSetAccentColorEmptyResetsToNeutral(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}), WithAccentColor("#FF0000"))
	d.SetAccentColor("")
	if got := d.AccentColor(); got != "" {
		t.Fatalf("AccentColor() = %q, want empty (neutral) after reset", got)
	}
}

// TestDefaultDropdownUncolored verifies that an unstyled dropdown (no accent,
// no custom styles) renders its open panel without any accent color: a plain
// border and a reverse-video highlight rather than an accent background.
func TestDefaultDropdownUncolored(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}))
	d, _ = d.doOpen()

	panel := d.list.View()
	if got := fgSeq(t, accentColor); strings.Contains(panel, got) {
		t.Fatalf("default panel should not use accent fg %q\n%q", got, panel)
	}
	if got := bgSeq(t, accentColor); strings.Contains(panel, got) {
		t.Fatalf("default panel should not use accent bg %q\n%q", got, panel)
	}
	// The highlighted row should be shown with reverse video.
	if !strings.Contains(panel, "\x1b[7m") {
		t.Fatalf("default highlighted row should use reverse video\n%q", panel)
	}
}

// TestDefaultFocusedArrowUncolored verifies the focused arrow on an unstyled
// trigger is emphasized with bold but carries no accent color.
func TestDefaultFocusedArrowUncolored(t *testing.T) {
	d := New(WithOptions([]string{"a", "b", "c"}))
	d.SetFocused(true)

	view := d.TriggerView()
	if got := fgSeq(t, accentColor); strings.Contains(view, got) {
		t.Fatalf("default focused arrow should not use accent fg %q\n%q", got, view)
	}
	if !strings.Contains(view, "\x1b[1m") {
		t.Fatalf("default focused arrow should be bold\n%q", view)
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

func TestTriggerArrowMatchesCustomStyleColor(t *testing.T) {
	green := "#2ECC71"
	accent := "#FF0000"
	d := New(
		WithOptions([]string{"Red", "Green", "Blue"}),
		WithInitialIndex(2),
		WithAccentColor(accent),
		WithTriggerStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)).
			Bold(true)),
	)
	d.SetFocused(true)

	view := d.TriggerView()
	greenSeq := fgSeq(t, green)
	accentSeq := fgSeq(t, accent)

	// The arrow (and the whole trigger) should use the custom trigger color.
	if !strings.Contains(view, greenSeq) {
		t.Fatalf("focused custom trigger should use the trigger color %q\n%q", greenSeq, view)
	}
	// Even with an explicit accent set, the custom trigger style must win so the
	// arrow does not clash with the caller's styling.
	if strings.Contains(view, accentSeq) {
		t.Fatalf("focused custom trigger arrow must not use accent color %q (should match trigger style)\n%q", accentSeq, view)
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
