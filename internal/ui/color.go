package ui

// ANSI escape sequences for the Outpost color palette.
const (
	reset        = "\033[0m"
	boldCode     = "\033[1m"
	dimCode      = "\033[2m"
	redCode      = "\033[31m"
	greenCode    = "\033[32m"
	yellowCode   = "\033[33m"
	blueCode     = "\033[34m"
	magentaCode  = "\033[35m"
	cyanCode     = "\033[36m"
	whiteCode    = "\033[37m"
	brightYellow = "\033[93m"
)

// Reset is the ANSI reset sequence.
const Reset = reset

// wrap returns s wrapped in the given ANSI code, or unchanged when color is off.
func wrap(code, s string) string {
	if !ColorEnabled {
		return s
	}
	return code + s + reset
}

// Amber applies the brand/identity color (ANSI yellow).
func Amber(s string) string { return wrap(yellowCode, s) }

// Green applies the success/complete color.
func Green(s string) string { return wrap(greenCode, s) }

// Cyan applies the active/in-progress color.
func Cyan(s string) string { return wrap(cyanCode, s) }

// Orange applies the warning/caution color (ANSI bright yellow).
func Orange(s string) string { return wrap(brightYellow, s) }

// Red applies the error/destructive/failed color.
func Red(s string) string { return wrap(redCode, s) }

// Purple applies the Claude/AI activity color (ANSI magenta).
func Purple(s string) string { return wrap(magentaCode, s) }

// Blue applies the actionable commands/hints color.
func Blue(s string) string { return wrap(blueCode, s) }

// White applies the primary content/values color.
func White(s string) string { return wrap(whiteCode, s) }

// Dim applies the chrome/labels/secondary color (ANSI faint).
func Dim(s string) string { return wrap(dimCode, s) }

// Bold wraps s in ANSI bold.
func Bold(s string) string { return wrap(boldCode, s) }
