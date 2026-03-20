package ui

import "strings"

// Table renders an aligned table with amber headers and an optional footer.
type Table struct {
	headers []string
	rows    [][]string
	footer  []string
}

// NewTable creates a table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// Row adds a data row. Column count should match headers.
func (t *Table) Row(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Footer sets the footer items, rendered as dim text with · separators.
func (t *Table) Footer(items ...string) {
	t.footer = items
}

// Render writes the table to stderr.
func (t *Table) Render() {
	widths := t.columnWidths()
	const gutter = 2

	// Headers (amber).
	var hdr strings.Builder
	hdr.WriteString("  ")
	for i, h := range t.headers {
		if i < len(t.headers)-1 {
			hdr.WriteString(PadRight(Amber(h), widths[i]+gutter))
		} else {
			hdr.WriteString(Amber(h))
		}
	}
	Errln(hdr.String())

	// Data rows.
	for _, row := range t.rows {
		var line strings.Builder
		line.WriteString("  ")
		for i, cell := range row {
			if i < len(row)-1 && i < len(widths) {
				line.WriteString(PadRight(cell, widths[i]+gutter))
			} else {
				line.WriteString(cell)
			}
		}
		Errln(line.String())
	}

	// Footer.
	if len(t.footer) > 0 {
		Errln()
		Errln("  " + Dim(strings.Join(t.footer, "  \u00b7  ")))
	}
}

// columnWidths returns the max visual width for each column.
func (t *Table) columnWidths() []int {
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = StringWidth(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				if w := StringWidth(cell); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}
	return widths
}
