package service

import "strings"

var duplicates = map[rune]rune{
	'а': 'a',
	'в': 'b',
	'е': 'e',
	'к': 'k',
	'м': 'm',
	'о': 'o',
	'р': 'p',
	'с': 'c',
	'т': 't',
	'у': 'y',
	'х': 'x',
}

func AlphabetFix(incoming string) string {
	lowerCased := strings.ToLower(incoming)
	builder := strings.Builder{}

	for _, char := range lowerCased {
		charReplacement, ok := duplicates[char]
		if ok {
			builder.WriteRune(charReplacement)
		} else {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
