package domain

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// DefaultSlugFallback is used when slugifying an empty or non-alphanumeric string.
const DefaultSlugFallback = "post"

// Slugify converts a title into a lowercase, ASCII-safe, hyphen-separated slug.
// Accented unicode characters are decomposed and stripped of diacritics.
// Any sequence of non-alphanumeric characters is replaced with a single hyphen.
// Leading and trailing hyphens are removed. If the resulting string is empty,
// DefaultSlugFallback ("post") is returned.
func Slugify(title string) string {
	// 1. Decompose unicode characters (e.g., 'é' -> 'e' + combining accent) and remove diacritics
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, err := transform.String(t, title)
	if err != nil {
		normalized = title
	}

	// 2. Convert to lowercase and replace non-alphanumeric characters with hyphens
	normalized = strings.ToLower(normalized)
	var b strings.Builder
	b.Grow(len(normalized))

	lastWasHyphen := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastWasHyphen = false
		} else if !lastWasHyphen {
			b.WriteByte('-')
			lastWasHyphen = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return DefaultSlugFallback
	}
	return slug
}

// AppendSlugSuffix appends a numeric suffix to a slug (e.g. "my-post", 2 -> "my-post-2").
// If slug is empty, DefaultSlugFallback is used as base.
func AppendSlugSuffix(slug string, suffix int) string {
	if slug == "" {
		slug = DefaultSlugFallback
	}
	return fmt.Sprintf("%s-%d", slug, suffix)
}
