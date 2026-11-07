package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/controller"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyChatService struct {
	controller.ChatServiceContract
	sendMessageFn func(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error)
}

func (d *dummyChatService) SendMessage(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error) {
	if d.sendMessageFn != nil {
		return d.sendMessageFn(ctx, conversationID, req, files)
	}
	return model.SendMessageResponse{}, nil
}

func setupChatTestApp(ctl controller.ChatController) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: config.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))),
	})
	app.Post("/workspaces/:wid/agent/chat/conversations/:id/messages", ctl.SendMessage)
	return app
}

func sendChatMessageRequest(app *fiber.App, wid, convID int64, fieldName, filename string, content []byte, textContent string) (*http.Response, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if textContent != "" {
		if err := writer.WriteField("content", textContent); err != nil {
			return nil, err
		}
	}

	if filename != "" {
		part, err := writer.CreateFormFile(fieldName, filename)
		if err != nil {
			return nil, err
		}
		if len(content) > 0 {
			if _, err := part.Write(content); err != nil {
				return nil, err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d/messages", wid, convID), &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return app.Test(req)
}

func TestChatController_SendMessage_AttachmentLimit(t *testing.T) {
	allowed := []string{".png", ".jpg", ".jpeg", ".webp", ".pdf"}

	t.Run("attachment exceeding 500KB returns 413 FILE_TOO_LARGE", func(t *testing.T) {
		svcCalled := false
		svc := &dummyChatService{
			sendMessageFn: func(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error) {
				svcCalled = true
				return model.SendMessageResponse{}, nil
			},
		}

		ctl := controller.NewChatController(svc, 10, allowed)
		app := setupChatTestApp(ctl)

		// 500KB + 1 byte (512,001 bytes)
		oversized := make([]byte, controller.MaxChatAttachmentSize+1)
		copy(oversized, validPNGBytes)

		resp, err := sendChatMessageRequest(app, 1, 100, "file", "receipt.png", oversized, "Receipt")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)
		assert.False(t, svcCalled, "Service should not be invoked when attachment exceeds limit")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "FILE_TOO_LARGE")
		assert.Contains(t, string(body), "file size exceeds maximum allowed limit")
	})

	t.Run("attachment in files array exceeding 500KB returns 413 FILE_TOO_LARGE", func(t *testing.T) {
		svcCalled := false
		svc := &dummyChatService{
			sendMessageFn: func(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error) {
				svcCalled = true
				return model.SendMessageResponse{}, nil
			},
		}

		ctl := controller.NewChatController(svc, 10, allowed)
		app := setupChatTestApp(ctl)

		oversized := make([]byte, controller.MaxChatAttachmentSize+1)
		copy(oversized, validPNGBytes)

		resp, err := sendChatMessageRequest(app, 1, 100, "files", "receipt.png", oversized, "Receipt")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)
		assert.False(t, svcCalled)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "FILE_TOO_LARGE")
	})

	t.Run("attachment exactly 500KB passes controller size check", func(t *testing.T) {
		svcCalled := false
		svc := &dummyChatService{
			sendMessageFn: func(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error) {
				svcCalled = true
				assert.Len(t, files, 1)
				assert.Equal(t, controller.MaxChatAttachmentSize, files[0].Size)
				return model.SendMessageResponse{}, nil
			},
		}

		ctl := controller.NewChatController(svc, 10, allowed)
		app := setupChatTestApp(ctl)

		// Exactly 500KB (512,000 bytes)
		exact := make([]byte, controller.MaxChatAttachmentSize)
		copy(exact, validPNGBytes)

		resp, err := sendChatMessageRequest(app, 1, 100, "file", "receipt.png", exact, "Receipt")
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.True(t, svcCalled, "Service should be called for files <= 500KB")
	})
}
