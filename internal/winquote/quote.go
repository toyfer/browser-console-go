package winquote

import "strings"

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	return strings.ContainsAny(s, " \t\"")
}

func QuoteArg(s string) string {
	if !needsQuote(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	nSlash := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			nSlash++
		case '"':
			for j := 0; j < nSlash; j++ {
				b.WriteString(`\\`)
			}
			nSlash = 0
			b.WriteString(`\"`)
		default:
			for j := 0; j < nSlash; j++ {
				b.WriteByte('\\')
			}
			nSlash = 0
			b.WriteByte(s[i])
		}
	}
	for j := 0; j < nSlash; j++ {
		b.WriteString(`\\`)
	}
	b.WriteByte('"')
	return b.String()
}

func CommandLine(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = QuoteArg(a)
	}
	return strings.Join(parts, " ")
}
