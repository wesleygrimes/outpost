package ui

// Error prints a standalone error line to stderr.
//
//	✗ No server configured.
func Error(text string) {
	Errln("  " + Fail(text))
}

// Fix prints a standalone fix command to stderr.
//
//	outpost login <host:port> <token>
func Fix(text string) {
	Errln("  " + Blue(text))
}
