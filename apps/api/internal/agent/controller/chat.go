package controller

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// MaxChatAttachmentSize defines the maximum allowed chat attachment size (500KB = 512,000 bytes)
// to prevent upstream LLM provider timeouts and internal errors.
const MaxChatAttachmentSize int64 = 500 * 1024

// ChatServiceContract defines the operations required by ChatController.
type ChatServiceContract interface {
	CreateConversation(ctx context.Context, title string) (model.ConversationResponse, error)
	ListConversations(ctx context.Context, page, limit int) (model.ListConversationsResponse, error)
	GetConversation(ctx context.Context, id int64) (model.ConversationDetailResponse, error)
	DeleteConversation(ctx context.Context, id int64) error
	SendMessage(ctx context.Context, conversationID int64, req model.SendMessageRequest, files []model.AttachmentUpload) (model.SendMessageResponse, error)
	ConfirmDraft(ctx context.Context, messageID int64, editedProposal *model.DraftProposal) (int64, error)
	DiscardDraft(ctx context.Context, messageID int64) error
	GetAttachment(ctx context.Context, attachmentID int64) (model.AttachmentDownload, error)
}

// ChatController handles HTTP requests for agent chat conversations, messages, and attachments.
type ChatController struct {
	svc         ChatServiceContract
	maxSizeMB   int
	allowedExts map[string]bool
}

// NewChatController constructs a new ChatController backed by ChatServiceContract.
func NewChatController(svc ChatServiceContract, maxSizeMB int, allowedTypes []string) ChatController {
	return ChatController{
		svc:         svc,
		maxSizeMB:   maxSizeMB,
		allowedExts: computeAllowedExtensions(allowedTypes),
	}
}

// ListConversations lists chat conversations for the workspace.
//
//	@Summary		List chat conversations
//	@Description	Retrieve paginated list of chat conversations in the workspace
//	@Tags			Agent Chat
//	@Produce		json
//	@Param			wid		path		int	true	"Workspace ID"
//	@Param			page	query		int	false	"Page number (default 1)"
//	@Param			limit	query		int	false	"Page size limit (default 20)"
//	@Success		200		{object}	httpx.WebResponse[model.ListConversationsResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/conversations [get]
func (ctl ChatController) ListConversations(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	res, err := ctl.svc.ListConversations(c, page, limit)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// CreateConversation creates a new chat conversation.
//
//	@Summary		Create chat conversation
//	@Description	Create a new conversation session for Talk To Agent
//	@Tags			Agent Chat
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.CreateConversationRequest	false	"Conversation creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.ConversationResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/conversations [post]
func (ctl ChatController) CreateConversation(c fiber.Ctx) error {
	var req model.CreateConversationRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&req); err != nil {
			return err
		}
	}

	res, err := ctl.svc.CreateConversation(c, req.Title)
	if err != nil {
		return err
	}
	return httpx.Created(c, res)
}

// GetConversation retrieves a conversation and all its messages and attachments.
//
//	@Summary		Get chat conversation
//	@Description	Retrieve detailed view of a conversation including all messages and attachments
//	@Tags			Agent Chat
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Conversation ID"
//	@Success		200	{object}	httpx.WebResponse[model.ConversationDetailResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/conversations/{id} [get]
func (ctl ChatController) GetConversation(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	res, err := ctl.svc.GetConversation(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// DeleteConversation deletes a conversation and all its associated attachments.
//
//	@Summary		Delete chat conversation
//	@Description	Delete a conversation and remove all stored attachment files from storage
//	@Tags			Agent Chat
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Conversation ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/conversations/{id} [delete]
func (ctl ChatController) DeleteConversation(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	if err := ctl.svc.DeleteConversation(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// SendMessage sends a user message with optional file attachments to the agent.
//
//	@Summary		Send chat message
//	@Description	Post a user message with optional document or image attachments to a conversation
//	@Tags			Agent Chat
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			wid		path		int		true	"Workspace ID"
//	@Param			id		path		int		true	"Conversation ID"
//	@Param			content	formData	string	false	"Message content"
//	@Param			file	formData	file	false	"Attachment file (.png, .jpg, .jpeg, .webp, .pdf)"
//	@Param			files	formData	file	false	"Additional attachment files"
//	@Success		200		{object}	httpx.WebResponse[model.SendMessageResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		413		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		429		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Failure		502		{object}	httpx.ErrorResponse
//	@Failure		504		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/conversations/{id}/messages [post]
func (ctl ChatController) SendMessage(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	content := c.FormValue("content")

	form, formErr := c.MultipartForm()
	var uploads []model.AttachmentUpload

	if formErr == nil && form != nil {
		var allFiles []*multipart.FileHeader
		if fhs, ok := form.File["file"]; ok {
			allFiles = append(allFiles, fhs...)
		}
		if fhs, ok := form.File["files"]; ok {
			allFiles = append(allFiles, fhs...)
		}

		for _, fh := range allFiles {
			if fh == nil {
				continue
			}
			if fh.Size <= 0 {
				return coredomain.ErrFileEmpty
			}
			if fh.Size > MaxChatAttachmentSize || (ctl.maxSizeMB > 0 && fh.Size > int64(ctl.maxSizeMB)*1024*1024) {
				return domain.ErrFileTooLarge
			}

			ext := strings.ToLower(filepath.Ext(fh.Filename))
			if !ctl.allowedExts[ext] {
				return domain.ErrUnsupportedDocument
			}

			src, err := fh.Open()
			if err != nil {
				return err
			}

			sniffBuf := make([]byte, 512)
			n, _ := io.ReadFull(src, sniffBuf)
			sniffedMime := http.DetectContentType(sniffBuf[:n])
			if idx := strings.Index(sniffedMime, ";"); idx != -1 {
				sniffedMime = strings.TrimSpace(sniffedMime[:idx])
			}
			if !analyzableMIMEs[sniffedMime] {
				_ = src.Close()
				return domain.ErrUnsupportedDocument
			}

			if _, err := src.Seek(0, io.SeekStart); err != nil {
				_ = src.Close()
				return err
			}
			data, err := io.ReadAll(src)
			_ = src.Close()
			if err != nil {
				return err
			}

			uploads = append(uploads, model.AttachmentUpload{
				FileName: fh.Filename,
				MIME:     sniffedMime,
				Size:     fh.Size,
				Data:     data,
			})
		}
	}

	res, err := ctl.svc.SendMessage(c, id, model.SendMessageRequest{Content: content}, uploads)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// ConfirmDraft confirms a proposed financial draft, saving it as a draft transaction.
//
//	@Summary		Confirm draft proposal
//	@Description	Confirm and save a draft transaction from an agent proposal, with optional user edits
//	@Tags			Agent Chat
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Message ID"
//	@Param			request	body		model.ConfirmDraftRequest	false	"Optional edited proposal payload"
//	@Success		200		{object}	httpx.WebResponse[model.ConfirmDraftResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/messages/{id}/confirm [post]
func (ctl ChatController) ConfirmDraft(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	var req model.ConfirmDraftRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().Body(&req); err != nil {
			return err
		}
	}

	txID, err := ctl.svc.ConfirmDraft(c, id, req.Proposal)
	if err != nil {
		return err
	}
	return httpx.OK(c, model.ConfirmDraftResponse{TransactionID: txID})
}

// DiscardDraft marks a proposed draft message as discarded.
//
//	@Summary		Discard draft proposal
//	@Description	Discard an agent draft proposal without creating any transaction
//	@Tags			Agent Chat
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Message ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		422	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/messages/{id}/discard [post]
func (ctl ChatController) DiscardDraft(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	if err := ctl.svc.DiscardDraft(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// DownloadAttachment downloads the content stream of an attachment file.
//
//	@Summary		Download chat attachment
//	@Description	Stream download a chat attachment file
//	@Tags			Agent Chat
//	@Produce		application/octet-stream
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Attachment ID"
//	@Success		200	{file}		binary
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/agent/chat/attachments/{id}/download [get]
func (ctl ChatController) DownloadAttachment(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	download, err := ctl.svc.GetAttachment(c, id)
	if err != nil {
		return err
	}
	if download.Body != nil {
		defer func() { _ = download.Body.Close() }()
	}

	data, err := io.ReadAll(download.Body)
	if err != nil {
		return err
	}

	contentType := download.MIME
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	safeName := strings.ReplaceAll(download.FileName, `"`, "")
	c.Set(fiber.HeaderContentType, contentType)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, safeName))
	return c.Status(fiber.StatusOK).Send(data)
}
