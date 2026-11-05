package domain

import (
	"strings"

	"golang.org/x/text/language"
)

// NormalizeLocale parses and canonicalizes a BCP 47 locale tag (e.g. "en-us" -> "en-US", "id" -> "id").
func NormalizeLocale(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrInvalidLocale
	}

	tag, err := language.Parse(trimmed)
	if err != nil {
		return "", ErrInvalidLocale
	}

	return tag.String(), nil
}

// IsLocaleSupported checks if the given normalized locale is in the supported locales list.
func IsLocaleSupported(locale string, supportedLocales []string) bool {
	for _, s := range supportedLocales {
		if strings.EqualFold(locale, s) {
			return true
		}
	}
	return false
}
