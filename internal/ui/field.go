package ui

// Field prints a labeled field to stderr (standalone, outside a checklist).
//
//	Branch     wes/premium-healthchecks
func Field(label, value string) {
	Errf("  %s%s\n", Dim(PadRight(label, 14)), value)
}
