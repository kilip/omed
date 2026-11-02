package domain_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/base"
	"github.com/stretchr/testify/assert"
)

func TestFinancialAgentErrorsRegistered(t *testing.T) {
	testCases := []struct {
		err            error
		expectedCode   string
		expectedStatus int
		expectedMsg    string
	}{
		{
			err:            domain.ErrFinancialAgentDisabled,
			expectedCode:   "FINANCIAL_AGENT_DISABLED",
			expectedStatus: fiber.StatusConflict,
			expectedMsg:    "financial agent is disabled for this workspace",
		},
		{
			err:            domain.ErrFinancialAgentNoProvider,
			expectedCode:   "FINANCIAL_AGENT_NO_PROVIDER",
			expectedStatus: fiber.StatusUnprocessableEntity,
			expectedMsg:    "no suitable provider configured for financial agent",
		},
		{
			err:            domain.ErrUnsupportedDocument,
			expectedCode:   "UNSUPPORTED_DOCUMENT",
			expectedStatus: fiber.StatusUnprocessableEntity,
			expectedMsg:    "unsupported document format for this provider",
		},
		{
			err:            domain.ErrAnalysisRateLimited,
			expectedCode:   "RATE_LIMITED",
			expectedStatus: fiber.StatusTooManyRequests,
			expectedMsg:    "financial analysis rate limit exceeded",
		},
		{
			err:            domain.ErrAnalysisFailed,
			expectedCode:   "FINANCIAL_ANALYSIS_FAILED",
			expectedStatus: fiber.StatusBadGateway,
			expectedMsg:    "financial analysis failed",
		},
		{
			err:            domain.ErrAnalysisTimeout,
			expectedCode:   "FINANCIAL_ANALYSIS_TIMEOUT",
			expectedStatus: fiber.StatusGatewayTimeout,
			expectedMsg:    "financial analysis timed out",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedCode, func(t *testing.T) {
			item := base.MapError(tc.err)
			assert.Equal(t, tc.expectedCode, item.Code)
			assert.Equal(t, tc.expectedStatus, item.Status)
			assert.Equal(t, tc.expectedMsg, item.Message)
		})
	}
}
