package ui

// Header prints the brand header to stderr.
//
//	OUTPOST v0.1.0  Provisioning myserver.grimes.pro
func Header(context string) {
	brand := Amber("OUTPOST") + " " + Dim(Version)
	if context != "" {
		brand += "  " + context
	}
	Errf("\n  %s\n", brand)
}
