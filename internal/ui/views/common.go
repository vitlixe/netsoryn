package views

// ContentSizeMsg is sent by the root model to notify views of their available drawing area.
type ContentSizeMsg struct {
	Width  int
	Height int
}
