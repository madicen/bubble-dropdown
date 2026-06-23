package bubbledropdown

// ItemChosenMsg is sent when the user selects an item from the dropdown
// (Enter key or mouse click on an option). Handle it in your root model;
// call SetSelectedIndex(msg.Index) on the dropdown to update displayed value.
type ItemChosenMsg struct {
	Index int    // zero-based index into the options slice
	Value string // the option string at that index
}

// ItemCanceledMsg is sent when the user dismisses the dropdown without
// selecting an item (Esc key or click outside the panel).
type ItemCanceledMsg struct{}
