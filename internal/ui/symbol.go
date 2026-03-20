package ui

// Status symbols. Each pairs with a specific color per the design system.
const (
	SymCheck = "\u2713" // Green - complete
	SymSpin  = "\u2838" // Cyan - running
	SymDot   = "\u25cf" // Green - done
	SymWait  = "\u25c9" // Purple - waiting
	SymWarn  = "\u26a0" // Orange - warning
	SymFail  = "\u2717" // Red - failed/error
)

// Symbol returns a colored symbol followed by text.
func Symbol(sym, text string) string {
	colored := colorForSymbol(sym)
	if text == "" {
		return colored
	}
	return colored + " " + text
}

// Check returns a green check mark followed by text.
func Check(text string) string { return Symbol(SymCheck, text) }

// Fail returns a red cross followed by text.
func Fail(text string) string { return Symbol(SymFail, text) }

// Warn returns an orange warning sign followed by text.
func Warn(text string) string { return Symbol(SymWarn, text) }

// Spin returns a cyan spinner frame followed by text.
func Spin(text string) string { return Symbol(SymSpin, text) }

// colorForSymbol applies the design-system color to a known symbol.
func colorForSymbol(sym string) string {
	switch sym {
	case SymCheck, SymDot:
		return Green(sym)
	case SymSpin:
		return Cyan(sym)
	case SymWait:
		return Purple(sym)
	case SymWarn:
		return Orange(sym)
	case SymFail:
		return Red(sym)
	default:
		return sym
	}
}
