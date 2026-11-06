package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentchatattachment"
	"github.com/kilip/omed/api/ent/agentchatconversation"
	"github.com/kilip/omed/api/ent/agentchatmessage"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/base"
)

// AppendMessageParams aliases model.AppendMessageParams for repository callers.
type AppendMessageParams = model.AppendMessageParams

// CreateAttachmentParams aliases model.CreateAttachmentParams for repository callers.
type CreateAttachmentParams = model.CreateAttachmentParams

// ChatRepository defines persistence operations for agent chat conversations, messages, and attachments.
type ChatRepository interface {
	CreateConversation(ctx context.Context, title string) (*ent.AgentChatConversation, error)
	ListConversations(ctx context.Context, page, limit int) ([]*ent.AgentChatConversation, int, error)
	GetConversation(ctx context.Context, id int64) (*ent.AgentChatConversation, error)
	GetConversationWithMessages(ctx context.Context, id int64) (*ent.AgentChatConversation, []*ent.AgentChatMessage, error)
	DeleteConversation(ctx context.Context, id int64) error
	AppendMessage(ctx context.Context, params AppendMessageParams) (*ent.AgentChatMessage, error)
	ListMessages(ctx context.Context, conversationID int64) ([]*ent.AgentChatMessage, error)
	GetMessage(ctx context.Context, id int64) (*ent.AgentChatMessage, error)
	UpdateDraftStatus(ctx context.Context, messageID int64, status agentchatmessage.DraftStatus, confirmedTxID *int64) (*ent.AgentChatMessage, error)
	CreateAttachment(ctx context.Context, params CreateAttachmentParams) (*ent.AgentChatAttachment, error)
	GetAttachment(ctx context.Context, id int64) (*ent.AgentChatAttachment, error)
	ListAttachmentsByConversation(ctx context.Context, conversationID int64) ([]*ent.AgentChatAttachment, error)
	ListAttachmentsByMessage(ctx context.Context, messageID int64) ([]*ent.AgentChatAttachment, error)
}

type chatRepository struct {
	cli *ent.Client
}

// NewChatRepository creates a new ChatRepository instance.
func NewChatRepository(cli *ent.Client) ChatRepository {
	return &chatRepository{cli: cli}
}

// CreateConversation creates a new conversation owned by the authenticated user in the current workspace.
func (r *chatRepository) CreateConversation(ctx context.Context, title string) (*ent.AgentChatConversation, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	builder := r.cli.AgentChatConversation.Create().
		SetWorkspaceID(wsID).
		SetCreatedBy(userID).
		SetUpdatedBy(userID)

	if title != "" {
		builder.SetTitle(title)
	}

	return builder.Save(ctx)
}

// ListConversations lists conversations owned by the authenticated user in the current workspace, paginated.
func (r *chatRepository) ListConversations(ctx context.Context, page, limit int) ([]*ent.AgentChatConversation, int, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, 0, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, 0, base.ErrUnauthorized
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	conversations, err := q.
		Order(ent.Desc(agentchatconversation.FieldUpdatedAt), ent.Desc(agentchatconversation.FieldID)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// GetConversation retrieves a single conversation owned by the authenticated user.
func (r *chatRepository) GetConversation(ctx context.Context, id int64) (*ent.AgentChatConversation, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	conv, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(id),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}

	return conv, nil
}

// GetConversationWithMessages retrieves a conversation along with its messages ordered chronologically.
func (r *chatRepository) GetConversationWithMessages(ctx context.Context, id int64) (*ent.AgentChatConversation, []*ent.AgentChatMessage, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, nil, base.ErrUnauthorized
	}

	conv, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(id),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		WithMessages(func(mq *ent.AgentChatMessageQuery) {
			mq.Where(agentchatmessage.WorkspaceIDEQ(wsID)).
				Order(ent.Asc(agentchatmessage.FieldCreatedAt), ent.Asc(agentchatmessage.FieldID))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, domain.ErrConversationNotFound
		}
		return nil, nil, err
	}

	messages := conv.Edges.Messages
	if messages == nil {
		messages = []*ent.AgentChatMessage{}
	}

	return conv, messages, nil
}

// DeleteConversation soft-deletes a conversation owned by the authenticated user.
func (r *chatRepository) DeleteConversation(ctx context.Context, id int64) error {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return base.ErrUnauthorized
	}

	affected, err := r.cli.AgentChatConversation.Update().
		Where(
			agentchatconversation.IDEQ(id),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		SetDeletedAt(time.Now()).
		SetDeletedByID(userID).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrConversationNotFound
	}

	return nil
}

// AppendMessage persists a new message in the conversation and refreshes conversation updated_at.
func (r *chatRepository) AppendMessage(ctx context.Context, params AppendMessageParams) (*ent.AgentChatMessage, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	convExists, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(params.ConversationID),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !convExists {
		return nil, domain.ErrConversationNotFound
	}

	builder := r.cli.AgentChatMessage.Create().
		SetWorkspaceID(wsID).
		SetConversationID(params.ConversationID).
		SetRole(params.Role).
		SetContent(params.Content).
		SetCreatedBy(userID).
		SetUpdatedBy(userID)

	if len(params.Parts) > 0 {
		builder.SetParts(params.Parts)
	}

	if params.DraftStatus != "" {
		builder.SetDraftStatus(params.DraftStatus)
	} else {
		builder.SetDraftStatus(agentchatmessage.DraftStatusNone)
	}

	if params.ConfirmedTransactionID != nil {
		builder.SetConfirmedTransactionID(*params.ConfirmedTransactionID)
	}

	msg, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}

	_ = r.cli.AgentChatConversation.UpdateOneID(params.ConversationID).
		SetUpdatedAt(time.Now()).
		Exec(ctx)

	return msg, nil
}

// ListMessages returns all messages for a conversation ordered chronologically.
func (r *chatRepository) ListMessages(ctx context.Context, conversationID int64) ([]*ent.AgentChatMessage, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	convExists, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(conversationID),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !convExists {
		return nil, domain.ErrConversationNotFound
	}

	return r.cli.AgentChatMessage.Query().
		Where(
			agentchatmessage.WorkspaceIDEQ(wsID),
			agentchatmessage.ConversationIDEQ(conversationID),
		).
		Order(ent.Asc(agentchatmessage.FieldCreatedAt), ent.Asc(agentchatmessage.FieldID)).
		All(ctx)
}

// UpdateDraftStatus updates the draft status and confirmed transaction reference of a proposed message.
func (r *chatRepository) UpdateDraftStatus(ctx context.Context, messageID int64, status agentchatmessage.DraftStatus, confirmedTxID *int64) (*ent.AgentChatMessage, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	msg, err := r.cli.AgentChatMessage.Query().
		Where(
			agentchatmessage.IDEQ(messageID),
			agentchatmessage.WorkspaceIDEQ(wsID),
			agentchatmessage.HasConversationWith(
				agentchatconversation.CreatedByEQ(userID),
				agentchatconversation.DeletedAtIsNil(),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, err
	}

	if msg.DraftStatus == agentchatmessage.DraftStatusConfirmed {
		return nil, domain.ErrDraftAlreadyConfirmed
	}
	if msg.DraftStatus == agentchatmessage.DraftStatusDiscarded {
		return nil, domain.ErrDraftAlreadyDiscarded
	}
	if (status == agentchatmessage.DraftStatusConfirmed || status == agentchatmessage.DraftStatusDiscarded) &&
		msg.DraftStatus != agentchatmessage.DraftStatusProposed {
		return nil, domain.ErrDraftNotProposed
	}

	updater := r.cli.AgentChatMessage.UpdateOneID(messageID).
		SetDraftStatus(status).
		SetUpdatedBy(userID)

	if confirmedTxID != nil {
		updater.SetConfirmedTransactionID(*confirmedTxID)
	} else {
		updater.ClearConfirmedTransactionID()
	}

	return updater.Save(ctx)
}

// CreateAttachment records metadata for a chat attachment.
func (r *chatRepository) CreateAttachment(ctx context.Context, params CreateAttachmentParams) (*ent.AgentChatAttachment, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	convExists, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(params.ConversationID),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !convExists {
		return nil, domain.ErrConversationNotFound
	}

	if params.MessageID != nil {
		msgExists, err := r.cli.AgentChatMessage.Query().
			Where(
				agentchatmessage.IDEQ(*params.MessageID),
				agentchatmessage.WorkspaceIDEQ(wsID),
				agentchatmessage.ConversationIDEQ(params.ConversationID),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !msgExists {
			return nil, domain.ErrMessageNotFound
		}
	}

	builder := r.cli.AgentChatAttachment.Create().
		SetWorkspaceID(wsID).
		SetConversationID(params.ConversationID).
		SetObjectKey(params.ObjectKey).
		SetBucket(params.Bucket).
		SetMime(params.MIME).
		SetSize(params.Size).
		SetFileName(params.FileName).
		SetCreatedBy(userID).
		SetUpdatedBy(userID)

	if params.MessageID != nil {
		builder.SetMessageID(*params.MessageID)
	}

	return builder.Save(ctx)
}

// ListAttachmentsByConversation returns all attachments linked to a conversation.
func (r *chatRepository) ListAttachmentsByConversation(ctx context.Context, conversationID int64) ([]*ent.AgentChatAttachment, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	convExists, err := r.cli.AgentChatConversation.Query().
		Where(
			agentchatconversation.IDEQ(conversationID),
			agentchatconversation.WorkspaceIDEQ(wsID),
			agentchatconversation.CreatedByEQ(userID),
			agentchatconversation.DeletedAtIsNil(),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !convExists {
		return nil, domain.ErrConversationNotFound
	}

	return r.cli.AgentChatAttachment.Query().
		Where(
			agentchatattachment.WorkspaceIDEQ(wsID),
			agentchatattachment.ConversationIDEQ(conversationID),
		).
		Order(ent.Asc(agentchatattachment.FieldCreatedAt), ent.Asc(agentchatattachment.FieldID)).
		All(ctx)
}

// ListAttachmentsByMessage returns all attachments linked to a message.
func (r *chatRepository) ListAttachmentsByMessage(ctx context.Context, messageID int64) ([]*ent.AgentChatAttachment, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	msgExists, err := r.cli.AgentChatMessage.Query().
		Where(
			agentchatmessage.IDEQ(messageID),
			agentchatmessage.WorkspaceIDEQ(wsID),
			agentchatmessage.HasConversationWith(
				agentchatconversation.CreatedByEQ(userID),
				agentchatconversation.DeletedAtIsNil(),
			),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !msgExists {
		return nil, domain.ErrMessageNotFound
	}

	return r.cli.AgentChatAttachment.Query().
		Where(
			agentchatattachment.WorkspaceIDEQ(wsID),
			agentchatattachment.MessageIDEQ(messageID),
		).
		Order(ent.Asc(agentchatattachment.FieldCreatedAt), ent.Asc(agentchatattachment.FieldID)).
		All(ctx)
}

// GetMessage retrieves a message owned by the authenticated user in the current workspace.
func (r *chatRepository) GetMessage(ctx context.Context, id int64) (*ent.AgentChatMessage, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	msg, err := r.cli.AgentChatMessage.Query().
		Where(
			agentchatmessage.IDEQ(id),
			agentchatmessage.WorkspaceIDEQ(wsID),
			agentchatmessage.HasConversationWith(
				agentchatconversation.CreatedByEQ(userID),
				agentchatconversation.DeletedAtIsNil(),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, err
	}
	return msg, nil
}

// GetAttachment retrieves an attachment owned by the authenticated user in the current workspace.
func (r *chatRepository) GetAttachment(ctx context.Context, id int64) (*ent.AgentChatAttachment, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}
	userID := base.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, base.ErrUnauthorized
	}

	att, err := r.cli.AgentChatAttachment.Query().
		Where(
			agentchatattachment.IDEQ(id),
			agentchatattachment.WorkspaceIDEQ(wsID),
			agentchatattachment.HasConversationWith(
				agentchatconversation.CreatedByEQ(userID),
				agentchatconversation.DeletedAtIsNil(),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrAttachmentNotFound
		}
		return nil, err
	}
	return att, nil
}
