package bubbledropdown

import (
	"strings"
	"testing"
)

func TestListModelSetStyles(t *testing.T) {
	m := newListModel([]string{"a", "b", "c"}, 0, 8, 1, defaultListStyles("62"))

	m.setStyles(defaultListStyles("#FF0000"))

	view := m.View()
	wantBorder := fgSeq(t, "#FF0000")
	wantCursor := bgSeq(t, "#FF0000")
	if !strings.Contains(view, wantBorder) {
		t.Fatalf("View() missing new border fg sequence %q\n%q", wantBorder, view)
	}
	if !strings.Contains(view, wantCursor) {
		t.Fatalf("View() missing new cursor bg sequence %q\n%q", wantCursor, view)
	}
}

func TestListModelSetStylesNilSafe(t *testing.T) {
	var m *listModel
	// Must not panic.
	m.setStyles(defaultListStyles("#FF0000"))
}
