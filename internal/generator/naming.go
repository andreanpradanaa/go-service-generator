package generator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Name holds every casing of one identifier the templates need, derived from
// a single user-supplied name such as "transfer-out" or "TransferOut".
type Name struct {
	Raw    string // as given
	Pascal string // TransferOut
	Camel  string // transferOut
	Snake  string // transfer_out
	Kebab  string // transfer-out
	Upper  string // TRANSFER_OUT
	Lower  string // transferout (valid Go package name)
	Title  string // Transfer Out
}

var namePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func NewName(raw string) (Name, error) {
	if !namePattern.MatchString(raw) {
		return Name{}, fmt.Errorf("invalid name %q: use letters, digits, '-' or '_', starting with a letter", raw)
	}

	words := splitWords(raw)
	lower := make([]string, len(words))
	pascal := make([]string, len(words))
	title := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
		pascal[i] = capitalize(lower[i])
		title[i] = pascal[i]
	}

	camel := lower[0] + strings.Join(pascal[1:], "")

	return Name{
		Raw:    raw,
		Pascal: strings.Join(pascal, ""),
		Camel:  camel,
		Snake:  strings.Join(lower, "_"),
		Kebab:  strings.Join(lower, "-"),
		Upper:  strings.ToUpper(strings.Join(lower, "_")),
		Lower:  strings.Join(lower, ""),
		Title:  strings.Join(title, " "),
	}, nil
}

// splitWords splits on '-', '_' and lower→upper case boundaries, so
// "transfer-out", "transfer_out" and "transferOut" all give [transfer out].
func splitWords(s string) []string {
	var (
		words []string
		cur   []rune
	)
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		if r == '-' || r == '_' {
			flush()
			continue
		}
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			flush()
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

// capitalize upper-cases the first letter, and fully upper-cases common
// initialisms so Go identifiers read naturally (e.g. "api" → "API").
func capitalize(w string) string {
	switch w {
	case "api", "id", "url", "http", "sso", "otp", "psp", "qr", "qris", "b2b", "b2b2c":
		return strings.ToUpper(w)
	}
	if w == "" {
		return w
	}
	return strings.ToUpper(w[:1]) + w[1:]
}
