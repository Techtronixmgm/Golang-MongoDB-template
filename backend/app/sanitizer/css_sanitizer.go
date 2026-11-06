package sanitizer

import (
	"basic-app/apperrors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.baoshuo.dev/cssparser"
)

var (
	unsafeCSSPattern = regexp.MustCompile(
		`(?i)(url\s*\(|expression\s*\(|javascript\s*:|vbscript\s*:|</?\s*(style|script)\b|@import\b)`,
	)

	allowedCSSAtRules = map[string]bool{
		"@media":             true,
		"@supports":          true,
		"@container":         true,
		"@layer":             true,
		"@keyframes":         true,
		"@-webkit-keyframes": true,
	}
)

// SanitizeCSS parses and validates a CSS stylesheet.
// It returns an error if the stylesheet contains unsupported or unsafe CSS.
func SanitizeCSS(raw string) (string, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return "", nil
	}

	stylesheet, err := cssparser.ParseStylesheet(raw)
	if err != nil {
		return "", fmt.Errorf("%w: malformed stylesheet", apperrors.ErrInvalidCSS)
	}

	for _, rule := range stylesheet.Rules {
		if err := validateCSSRule(rule); err != nil {
			return "", err
		}
	}

	return stylesheet.String(), nil
}

func validateCSSRule(rule *cssparser.CssRule) error {
	if rule == nil {
		return apperrors.ErrInvalidCSS
	}

	switch rule.Kind {
	case cssparser.QualifiedRule:
		for _, selector := range rule.Selectors {
			if containsUnsafeCSS(selector) {
				return apperrors.ErrInvalidCSS
			}
		}

		for _, declaration := range rule.Declarations {
			if err := validateCSSDeclaration(declaration); err != nil {
				return err
			}
		}

	case cssparser.AtRule:
		name := strings.ToLower(strings.TrimSpace(rule.Name))

		if !allowedCSSAtRules[name] {
			return fmt.Errorf(
				"%w: unsupported at-rule %s",
				apperrors.ErrInvalidCSS,
				name,
			)
		}

		if containsUnsafeCSS(rule.Prelude) {
			return apperrors.ErrInvalidCSS
		}

		for _, declaration := range rule.Declarations {
			if err := validateCSSDeclaration(declaration); err != nil {
				return err
			}
		}

		for _, nestedRule := range rule.Rules {
			if err := validateCSSRule(nestedRule); err != nil {
				return err
			}
		}

	default:
		return apperrors.ErrInvalidCSS
	}

	return nil
}

func validateCSSDeclaration(
	declaration *cssparser.Declaration,
) error {
	if declaration == nil {
		return apperrors.ErrInvalidCSS
	}

	property := strings.ToLower(
		strings.TrimSpace(declaration.Property),
	)

	switch property {
	case "behavior", "-moz-binding":
		return apperrors.ErrInvalidCSS
	}

	if containsUnsafeCSS(property) ||
		containsUnsafeCSS(declaration.Value) {
		return apperrors.ErrInvalidCSS
	}

	return nil
}

func containsUnsafeCSS(value string) bool {
	decoded := decodeCSSEscapes(value)
	return unsafeCSSPattern.MatchString(decoded)
}

// decodeCSSEscapes decodes CSS escape sequences before security checks.
func decodeCSSEscapes(value string) string {
	var result strings.Builder
	runes := []rune(value)

	for i := 0; i < len(runes); i++ {
		if runes[i] != '\\' {
			result.WriteRune(runes[i])
			continue
		}

		if i+1 >= len(runes) {
			result.WriteRune('\\')
			continue
		}

		next := runes[i+1]

		// CSS escaped newline.
		if next == '\n' || next == '\r' || next == '\f' {
			i++

			if next == '\r' && i+1 < len(runes) &&
				runes[i+1] == '\n' {
				i++
			}

			continue
		}

		// CSS hexadecimal escape.
		if isCSSHexDigit(next) {
			start := i + 1
			end := start

			for end < len(runes) &&
				end-start < 6 &&
				isCSSHexDigit(runes[end]) {
				end++
			}

			hex := string(runes[start:end])
			codePoint, err := strconv.ParseUint(hex, 16, 32)

			if err == nil &&
				codePoint > 0 &&
				codePoint <= utf8.MaxRune &&
				!(codePoint >= 0xD800 && codePoint <= 0xDFFF) {
				result.WriteRune(rune(codePoint))
			} else {
				result.WriteRune(utf8.RuneError)
			}

			i = end - 1

			// Consume optional whitespace after the escape.
			if i+1 < len(runes) && isCSSWhitespace(runes[i+1]) {
				i++

				if runes[i] == '\r' &&
					i+1 < len(runes) &&
					runes[i+1] == '\n' {
					i++
				}
			}

			continue
		}

		// Ordinary escaped character.
		result.WriteRune(next)
		i++
	}

	return strings.ToLower(result.String())
}

func isCSSHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func isCSSWhitespace(r rune) bool {
	return r == ' ' ||
		r == '\t' ||
		r == '\n' ||
		r == '\r' ||
		r == '\f'
}
