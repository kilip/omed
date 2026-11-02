package controller

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

var analyzableExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".pdf":  true,
}

var analyzableMIMEs = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/webp":      true,
	"application/pdf": true,
}

func computeAllowedExtensions(configured []string) map[string]bool {
	allowed := make(map[string]bool)
	if len(configured) == 0 {
		for ext := range analyzableExtensions {
			allowed[ext] = true
		}
		return allowed
	}
	for _, t := range configured {
		ext := strings.ToLower(strings.TrimSpace(t))
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		if analyzableExtensions[ext] {
			allowed[ext] = true
		}
	}
	return allowed
}

// FinancialServiceContract defines the operations required by FinancialController.
type FinancialServiceContract interface {
	Status(ctx context.Context) (model.FinancialStatus, error)
	Analyze(ctx context.Context, req model.AnalyzeRequest) (model.FinancialSuggestion, error)
}

// FinancialController handles HTTP requests for financial agent operations.
type FinancialController struct {
	svc         FinancialServiceContract
	maxSizeMB   int
	allowedExts map[string]bool
}

// NewFinancialController constructs a new FinancialController backed by FinancialServiceContract.
func NewFinancialController(svc FinancialServiceContract, maxSizeMB int, allowedTypes []string) FinancialController {
	return FinancialController{
		svc:         svc,
		maxSizeMB:   maxSizeMB,
		allowedExts: computeAllowedExtensions(allowedTypes),
	}
}

// Status retrieves the availability status of the financial agent.
//
//	@Summary		Get financial agent status
//	@Description	Retrieve availability and readiness of the financial agent for the workspace
//	@Tags			Agent Financial
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.FinancialStatus]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/financial/status [get]
func (ctl FinancialController) Status(c fiber.Ctx) error {
	status, err := ctl.svc.Status(c)
	if err != nil {
		return err
	}
	return httpx.OK(c, status)
}

// Analyze extracts transaction data from an uploaded financial document.
//
//	@Summary		Analyze financial document
//	@Description	Analyze an uploaded invoice, receipt, or payment proof and extract suggested transaction data
//	@Tags			Agent Financial
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			file	formData	file	true	"Document file to analyze (.png, .jpg, .jpeg, .webp, .pdf)"
//	@Param			flow	formData	string	true	"Transaction flow (INCOME or EXPENSE)"	Enums(INCOME, EXPENSE)
//	@Success		200		{object}	httpx.WebResponse[model.FinancialSuggestion]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		429		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Failure		502		{object}	httpx.ErrorResponse
//	@Failure		504		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/financial/analyze [post]
func (ctl FinancialController) Analyze(c fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return coredomain.ErrFileRequired
	}
	if fh.Size <= 0 {
		return coredomain.ErrFileEmpty
	}
	if ctl.maxSizeMB > 0 && fh.Size > int64(ctl.maxSizeMB)*1024*1024 {
		return domain.ErrUnsupportedDocument
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !ctl.allowedExts[ext] {
		return domain.ErrUnsupportedDocument
	}

	flow := c.FormValue("flow")
	if flow != "INCOME" && flow != "EXPENSE" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "flow must be INCOME or EXPENSE")
	}

	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	sniffBuf := make([]byte, 512)
	n, _ := io.ReadFull(src, sniffBuf)
	sniffedMime := http.DetectContentType(sniffBuf[:n])
	if idx := strings.Index(sniffedMime, ";"); idx != -1 {
		sniffedMime = strings.TrimSpace(sniffedMime[:idx])
	}
	if !analyzableMIMEs[sniffedMime] {
		return domain.ErrUnsupportedDocument
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	suggestion, err := ctl.svc.Analyze(c, model.AnalyzeRequest{
		Flow:     flow,
		FileName: fh.Filename,
		MIME:     sniffedMime,
		Data:     data,
	})
	if err != nil {
		return err
	}

	return httpx.OK(c, suggestion)
}
