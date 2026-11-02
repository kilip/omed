package controller_test

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/controller"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/agent/service/mock"
	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	validPNGBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	validPDFBytes = []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj\ntrailer\n<<\n/Root 1 0 R\n>>\n%%EOF")
)

func setupTestApp(ctl controller.FinancialController) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: config.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	app.Get("/workspaces/:wid/agent/financial/status", ctl.Status)
	app.Post("/workspaces/:wid/agent/financial/analyze", ctl.Analyze)
	return app
}

func sendAnalyzeRequest(app *fiber.App, wid int64, filename string, content []byte, flow *string, omitFile bool) (*http.Response, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if !omitFile {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}
		if len(content) > 0 {
			if _, err := part.Write(content); err != nil {
				return nil, err
			}
		}
	}

	if flow != nil {
		if err := writer.WriteField("flow", *flow); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/agent/financial/analyze", wid), &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return app.Test(req)
}

func TestFinancialController_Status_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Status(gomock.Any()).Return(model.FinancialStatus{
		Enabled:   true,
		Available: true,
	}, nil)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	req := httptest.NewRequest(fiber.MethodGet, "/workspaces/1/agent/financial/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestFinancialController_Status_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Status(gomock.Any()).Return(model.FinancialStatus{}, domain.ErrFinancialAgentDisabled)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	req := httptest.NewRequest(fiber.MethodGet, "/workspaces/1/agent/financial/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestFinancialController_Analyze_MissingFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	// Service should NEVER be called
	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "", nil, &flow, true)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // coredomain.ErrFileRequired maps to 400
}

func TestFinancialController_Analyze_EmptyFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	// Service should NEVER be called
	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "test.png", []byte{}, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode) // coredomain.ErrFileEmpty maps to 422
}

func TestFinancialController_Analyze_FileOverMaxSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	// maxSizeMB = 1 (1MB limit), provide 2MB bytes with PNG prefix
	oversized := make([]byte, 2*1024*1024)
	copy(oversized, validPNGBytes)

	ctl := controller.NewFinancialController(mockSvc, 1, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "large.png", oversized, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_InvalidFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "INVALID_FLOW"
	resp, err := sendAnalyzeRequest(app, 1, "test.png", validPNGBytes, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_MissingFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	resp, err := sendAnalyzeRequest(app, 1, "test.png", validPNGBytes, nil, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_SniffedMIMEMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	// Named x.png but content is plain text
	flow := "EXPENSE"
	textContent := []byte("this is plain text content, not a PNG image")
	resp, err := sendAnalyzeRequest(app, 1, "x.png", textContent, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_ExecutableNamedPNG_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	// Named x.png but content is ELF binary
	flow := "EXPENSE"
	elfContent := []byte("\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	resp, err := sendAnalyzeRequest(app, 1, "x.png", elfContent, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_CSVAndXLSXRejectedEvenWhenConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	// .csv and .xlsx configured in allowedTypes
	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf", ".csv", ".xlsx"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	csvContent := []byte("date,amount,description\n2026-01-01,100,Office\n")
	resp, err := sendAnalyzeRequest(app, 1, "report.csv", csvContent, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)

	resp, err = sendAnalyzeRequest(app, 1, "report.xlsx", []byte("PK\x03\x04test"), &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_AllowlistIntersection(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Any()).Times(0)

	// Configured allowedTypes only includes .pdf (intersection is only .pdf)
	ctl := controller.NewFinancialController(mockSvc, 10, []string{".pdf"})
	app := setupTestApp(ctl)

	// Valid PNG file should be rejected because .png is not in configured allowedTypes
	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "receipt.png", validPNGBytes, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestFinancialController_Analyze_PNGNamedPDF_SniffedAsPNG(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	// Expect service to be called with MIME "image/png" despite filename being x.pdf
	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Cond(func(x any) bool {
		req, ok := x.(model.AnalyzeRequest)
		if !ok {
			return false
		}
		return req.FileName == "x.pdf" &&
			req.MIME == "image/png" &&
			req.Flow == "EXPENSE" &&
			bytes.Equal(req.Data, validPNGBytes)
	})).Return(model.FinancialSuggestion{
		DocumentType: "receipt",
	}, nil)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "x.pdf", validPNGBytes, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestFinancialController_Analyze_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Cond(func(x any) bool {
		req, ok := x.(model.AnalyzeRequest)
		if !ok {
			return false
		}
		return req.FileName == "invoice.png" &&
			req.MIME == "image/png" &&
			req.Flow == "INCOME" &&
			bytes.Equal(req.Data, validPNGBytes)
	})).Return(model.FinancialSuggestion{
		DocumentType: "invoice",
	}, nil)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "INCOME"
	resp, err := sendAnalyzeRequest(app, 1, "invoice.png", validPNGBytes, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestFinancialController_Analyze_PDF_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockFinancialService(ctrl)

	mockSvc.EXPECT().Analyze(gomock.Any(), gomock.Cond(func(x any) bool {
		req, ok := x.(model.AnalyzeRequest)
		if !ok {
			return false
		}
		return req.FileName == "invoice.pdf" &&
			req.MIME == "application/pdf" &&
			req.Flow == "EXPENSE" &&
			bytes.Equal(req.Data, validPDFBytes)
	})).Return(model.FinancialSuggestion{
		DocumentType: "invoice",
	}, nil)

	ctl := controller.NewFinancialController(mockSvc, 10, []string{".png", ".pdf"})
	app := setupTestApp(ctl)

	flow := "EXPENSE"
	resp, err := sendAnalyzeRequest(app, 1, "invoice.pdf", validPDFBytes, &flow, false)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
}

