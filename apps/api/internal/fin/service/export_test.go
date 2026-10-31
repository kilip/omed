package service

import (
	"github.com/kilip/omed/api/internal/fin/domain"
)

// Export unexported functions for service_test.
func ResolveAccountRef(accounts []domain.Account, ref string) (domain.Account, error) {
	return resolveAccountRef(accounts, ref)
}

func TruncateText(s string, maxRunes int) string {
	return truncateText(s, maxRunes)
}
