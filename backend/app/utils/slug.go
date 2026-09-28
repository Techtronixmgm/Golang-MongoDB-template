package utils

import (
	"strings"
	"unicode"
)

func Slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	var builder strings.Builder
	lastWasHyphen := false

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastWasHyphen = false
			continue
		}

		if !lastWasHyphen && builder.Len() > 0 {
			builder.WriteByte('-')
			lastWasHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
