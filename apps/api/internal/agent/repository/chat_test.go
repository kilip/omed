package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentchatmessage"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chatTestFixture struct {
	cli      *ent.Client
	repo     repository.ChatRepository
	user1    *ent.User
	user2    *ent.User
	ctxUser1 context.Context // WS1, User 1
	ctxUser2 context.Context // WS1, User 2
	ctxWs2   context.Context // WS2, User 1
	ws1ID    int64
	ws2ID    int64
}

func setupChatRepo(t *testing.T) chatTestFixture {
	t.Helper()
	cli := testutil.CreateTestDB(fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1&_busy_timeout=5000", t.Name()))
	core.InitEntCli(cli)
	t.Cleanup(func() {
		_ = cli.Close()
	})

	initCtx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))

	u1, err := cli.User.Create().
		SetName("User One").
		SetEmail(fmt.Sprintf("u1_%s@example.com", t.Name())).
		Save(initCtx)
	require.NoError(t, err)

	u2, err := cli.User.Create().
		SetName("User Two").
		SetEmail(fmt.Sprintf("u2_%s@example.com", t.Name())).
		Save(initCtx)
	require.NoError(t, err)

	user1Ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u1.ID,
		Name: u1.Name,
	})

	user2Ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u2.ID,
		Name: u2.Name,
	})

	ws1, err := cli.Workspace.Create().
		SetName("Workspace 1").
		SetOwnerID(u1.ID).
		Save(user1Ctx)
	require.NoError(t, err)

	ws2, err := cli.Workspace.Create().
		SetName("Workspace 2").
		SetOwnerID(u1.ID).
		Save(user1Ctx)
	require.NoError(t, err)

	ctxUser1 := base.WithWorkspaceID(user1Ctx, ws1.ID)
	ctxUser2 := base.WithWorkspaceID(user2Ctx, ws1.ID)
	ctxWs2 := base.WithWorkspaceID(user1Ctx, ws2.ID)

	return chatTestFixture{
		cli:      cli,
		repo:     repository.NewChatRepository(cli),
		user1:    u1,
		user2:    u2,
		ctxUser1: ctxUser1,
		ctxUser2: ctxUser2,
		ctxWs2:   ctxWs2,
		ws1ID:    ws1.ID,
		ws2ID:    ws2.ID,
	}
}

func TestChat_CreateConversation(t *testing.T) {
	f := setupChatRepo(t)

	// Create with title
	conv1, err := f.repo.CreateConversation(f.ctxUser1, "Receipt Analysis")
	require.NoError(t, err)
	assert.NotEmpty(t, conv1.ID)
	assert.Equal(t, "Receipt Analysis", conv1.Title)
	assert.Equal(t, f.ws1ID, conv1.WorkspaceID)
	assert.Equal(t, f.user1.ID, conv1.CreatedBy)

	// Create without title
	conv2, err := f.repo.CreateConversation(f.ctxUser1, "")
	require.NoError(t, err)
	assert.NotEmpty(t, conv2.ID)
	assert.Empty(t, conv2.Title)
}

func TestChat_ListConversations(t *testing.T) {
	f := setupChatRepo(t)

	// Create 3 conversations for User 1 in WS 1
	c1, err := f.repo.CreateConversation(f.ctxUser1, "Conv 1")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	c2, err := f.repo.CreateConversation(f.ctxUser1, "Conv 2")
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	c3, err := f.repo.CreateConversation(f.ctxUser1, "Conv 3")
	require.NoError(t, err)

	// Create 1 conversation for User 2 in WS 1
	_, err = f.repo.CreateConversation(f.ctxUser2, "User 2 Conv")
	require.NoError(t, err)

	// Create 1 conversation for User 1 in WS 2
	_, err = f.repo.CreateConversation(f.ctxWs2, "WS 2 Conv")
	require.NoError(t, err)

	// List page 1, limit 2 for User 1 in WS 1
	items, total, err := f.repo.ListConversations(f.ctxUser1, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 2)
	assert.Equal(t, c3.ID, items[0].ID)
	assert.Equal(t, c2.ID, items[1].ID)

	// List page 2, limit 2
	itemsPage2, total2, err := f.repo.ListConversations(f.ctxUser1, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total2)
	require.Len(t, itemsPage2, 1)
	assert.Equal(t, c1.ID, itemsPage2[0].ID)

	// User 2 in WS 1 sees only their own conversation
	u2Items, u2Total, err := f.repo.ListConversations(f.ctxUser2, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, u2Total)
	require.Len(t, u2Items, 1)
	assert.Equal(t, "User 2 Conv", u2Items[0].Title)

	// Soft delete c2, verify it is hidden from List
	err = f.repo.DeleteConversation(f.ctxUser1, c2.ID)
	require.NoError(t, err)

	itemsAfterDelete, totalAfterDelete, err := f.repo.ListConversations(f.ctxUser1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, totalAfterDelete)
	require.Len(t, itemsAfterDelete, 2)
	assert.Equal(t, c3.ID, itemsAfterDelete[0].ID)
	assert.Equal(t, c1.ID, itemsAfterDelete[1].ID)
}

func TestChat_GetConversation(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Test Conv")
	require.NoError(t, err)

	// Success
	got, err := f.repo.GetConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, got.ID)
	assert.Equal(t, "Test Conv", got.Title)

	// Nonexistent ID
	_, err = f.repo.GetConversation(f.ctxUser1, 999999)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// User isolation: User 2 cannot access User 1's conversation
	_, err = f.repo.GetConversation(f.ctxUser2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// Workspace isolation: Querying from WS 2 returns not found
	_, err = f.repo.GetConversation(f.ctxWs2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// Soft deleted conversation returns not found
	err = f.repo.DeleteConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)

	_, err = f.repo.GetConversation(f.ctxUser1, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_GetConversationWithMessages(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Chat Flow")
	require.NoError(t, err)

	// Initially empty messages
	gotConv, msgs, err := f.repo.GetConversationWithMessages(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, gotConv.ID)
	assert.Empty(t, msgs)

	// Append messages
	m1, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "Receipt uploaded",
	})
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	m2, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Draft proposed",
		DraftStatus:    agentchatmessage.DraftStatusProposed,
	})
	require.NoError(t, err)

	// Fetch with messages - must be ordered chronologically
	gotConv, msgs, err = f.repo.GetConversationWithMessages(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, gotConv.ID)
	require.Len(t, msgs, 2)
	assert.Equal(t, m1.ID, msgs[0].ID)
	assert.Equal(t, m2.ID, msgs[1].ID)
	assert.Equal(t, "Receipt uploaded", msgs[0].Content)
	assert.Equal(t, "Draft proposed", msgs[1].Content)

	// Not found for other user
	_, _, err = f.repo.GetConversationWithMessages(f.ctxUser2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_DeleteConversation(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "To Delete")
	require.NoError(t, err)

	// User 2 cannot delete User 1's conversation
	err = f.repo.DeleteConversation(f.ctxUser2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// User 1 successfully soft deletes
	err = f.repo.DeleteConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)

	// Re-deleting returns ErrConversationNotFound
	err = f.repo.DeleteConversation(f.ctxUser1, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_UpdateConversationTitle(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Old Title")
	require.NoError(t, err)

	// User 2 cannot update User 1's conversation title
	err = f.repo.UpdateConversationTitle(f.ctxUser2, conv.ID, "Hacked Title")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// User 1 successfully updates title
	err = f.repo.UpdateConversationTitle(f.ctxUser1, conv.ID, "New Renamed Title")
	require.NoError(t, err)

	updated, err := f.repo.GetConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Renamed Title", updated.Title)

	// Updating non-existent conversation returns ErrConversationNotFound
	err = f.repo.UpdateConversationTitle(f.ctxUser1, 999999, "Another Title")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// Updating soft-deleted conversation returns ErrConversationNotFound
	err = f.repo.DeleteConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	err = f.repo.UpdateConversationTitle(f.ctxUser1, conv.ID, "After Delete Title")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_AppendMessage(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Conv")
	require.NoError(t, err)

	partsJSON := json.RawMessage(`{"type":"draft","proposal":{"kind":"expense","totalAmount":"70000.00"}}`)
	txID := int64(12345)

	msg, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID:         conv.ID,
		Role:                   agentchatmessage.RoleAssistant,
		Content:                "Proposed expense",
		Parts:                  partsJSON,
		DraftStatus:            agentchatmessage.DraftStatusProposed,
		ConfirmedTransactionID: &txID,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, conv.ID, msg.ConversationID)
	assert.Equal(t, agentchatmessage.RoleAssistant, msg.Role)
	assert.Equal(t, "Proposed expense", msg.Content)
	assert.JSONEq(t, string(partsJSON), string(msg.Parts))
	assert.Equal(t, agentchatmessage.DraftStatusProposed, msg.DraftStatus)
	require.NotNil(t, msg.ConfirmedTransactionID)
	assert.Equal(t, txID, *msg.ConfirmedTransactionID)

	// Cannot append to another user's conversation
	_, err = f.repo.AppendMessage(f.ctxUser2, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "Intruder",
	})
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// Cannot append to soft deleted conversation
	err = f.repo.DeleteConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)

	_, err = f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "Late message",
	})
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_ListMessages(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Conv")
	require.NoError(t, err)

	m1, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "First",
	})
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	m2, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Second",
	})
	require.NoError(t, err)

	msgs, err := f.repo.ListMessages(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, m1.ID, msgs[0].ID)
	assert.Equal(t, m2.ID, msgs[1].ID)

	// User 2 cannot list messages of User 1's conversation
	_, err = f.repo.ListMessages(f.ctxUser2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestChat_DraftStatusTransitions(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Draft Conv")
	require.NoError(t, err)

	// 1. Message without proposed draft (draft_status = none)
	mNone, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Plain reply",
		DraftStatus:    agentchatmessage.DraftStatusNone,
	})
	require.NoError(t, err)

	// Trying to confirm a non-proposed message -> ErrDraftNotProposed
	txID := int64(100)
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mNone.ID, agentchatmessage.DraftStatusConfirmed, &txID)
	assert.ErrorIs(t, err, domain.ErrDraftNotProposed)

	// Trying to discard a non-proposed message -> ErrDraftNotProposed
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mNone.ID, agentchatmessage.DraftStatusDiscarded, nil)
	assert.ErrorIs(t, err, domain.ErrDraftNotProposed)

	// 2. Proposed draft message -> Confirm flow
	mProposed1, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Proposed draft",
		DraftStatus:    agentchatmessage.DraftStatusProposed,
	})
	require.NoError(t, err)

	// Confirm successfully
	confirmedTxID := int64(200)
	confirmed, err := f.repo.UpdateDraftStatus(f.ctxUser1, mProposed1.ID, agentchatmessage.DraftStatusConfirmed, &confirmedTxID)
	require.NoError(t, err)
	assert.Equal(t, agentchatmessage.DraftStatusConfirmed, confirmed.DraftStatus)
	require.NotNil(t, confirmed.ConfirmedTransactionID)
	assert.Equal(t, confirmedTxID, *confirmed.ConfirmedTransactionID)

	// Confirming again -> ErrDraftAlreadyConfirmed
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mProposed1.ID, agentchatmessage.DraftStatusConfirmed, &confirmedTxID)
	assert.ErrorIs(t, err, domain.ErrDraftAlreadyConfirmed)

	// Discarding an already confirmed draft -> ErrDraftAlreadyConfirmed
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mProposed1.ID, agentchatmessage.DraftStatusDiscarded, nil)
	assert.ErrorIs(t, err, domain.ErrDraftAlreadyConfirmed)

	// 3. Proposed draft message -> Discard flow
	mProposed2, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Proposed draft 2",
		DraftStatus:    agentchatmessage.DraftStatusProposed,
	})
	require.NoError(t, err)

	// Discard successfully
	discarded, err := f.repo.UpdateDraftStatus(f.ctxUser1, mProposed2.ID, agentchatmessage.DraftStatusDiscarded, nil)
	require.NoError(t, err)
	assert.Equal(t, agentchatmessage.DraftStatusDiscarded, discarded.DraftStatus)
	assert.Nil(t, discarded.ConfirmedTransactionID)

	// Discarding again -> ErrDraftAlreadyDiscarded
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mProposed2.ID, agentchatmessage.DraftStatusDiscarded, nil)
	assert.ErrorIs(t, err, domain.ErrDraftAlreadyDiscarded)

	// Confirming an already discarded draft -> ErrDraftAlreadyDiscarded
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, mProposed2.ID, agentchatmessage.DraftStatusConfirmed, &txID)
	assert.ErrorIs(t, err, domain.ErrDraftAlreadyDiscarded)

	// 4. Nonexistent message ID -> ErrMessageNotFound
	_, err = f.repo.UpdateDraftStatus(f.ctxUser1, 999999, agentchatmessage.DraftStatusConfirmed, &txID)
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)

	// 5. User isolation: User 2 cannot update User 1's message draft status
	mProposed3, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleAssistant,
		Content:        "Proposed draft 3",
		DraftStatus:    agentchatmessage.DraftStatusProposed,
	})
	require.NoError(t, err)

	_, err = f.repo.UpdateDraftStatus(f.ctxUser2, mProposed3.ID, agentchatmessage.DraftStatusConfirmed, &txID)
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)
}

func TestChat_AttachmentOperations(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Attachments Conv")
	require.NoError(t, err)

	msg, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "Check receipts",
	})
	require.NoError(t, err)

	// Create attachment linked to message
	att1, err := f.repo.CreateAttachment(f.ctxUser1, repository.CreateAttachmentParams{
		ConversationID: conv.ID,
		MessageID:      &msg.ID,
		ObjectKey:      "workspaces/1/agent/chat/receipt1.jpg",
		Bucket:         "omed-bucket",
		MIME:           "image/jpeg",
		Size:           10240,
		FileName:       "receipt1.jpg",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, att1.ID)
	assert.Equal(t, conv.ID, att1.ConversationID)
	require.NotNil(t, att1.MessageID)
	assert.Equal(t, msg.ID, *att1.MessageID)
	assert.Equal(t, "receipt1.jpg", att1.FileName)

	// Create attachment linked to conversation without message
	att2, err := f.repo.CreateAttachment(f.ctxUser1, repository.CreateAttachmentParams{
		ConversationID: conv.ID,
		MessageID:      nil,
		ObjectKey:      "workspaces/1/agent/chat/invoice.pdf",
		Bucket:         "omed-bucket",
		MIME:           "application/pdf",
		Size:           51200,
		FileName:       "invoice.pdf",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, att2.ID)
	assert.Nil(t, att2.MessageID)

	// List attachments by conversation
	convAtts, err := f.repo.ListAttachmentsByConversation(f.ctxUser1, conv.ID)
	require.NoError(t, err)
	require.Len(t, convAtts, 2)
	assert.Equal(t, att1.ID, convAtts[0].ID)
	assert.Equal(t, att2.ID, convAtts[1].ID)

	// List attachments by message
	msgAtts, err := f.repo.ListAttachmentsByMessage(f.ctxUser1, msg.ID)
	require.NoError(t, err)
	require.Len(t, msgAtts, 1)
	assert.Equal(t, att1.ID, msgAtts[0].ID)

	// Create with nonexistent conversation -> ErrConversationNotFound
	_, err = f.repo.CreateAttachment(f.ctxUser1, repository.CreateAttachmentParams{
		ConversationID: 999999,
		ObjectKey:      "invalid.jpg",
		Bucket:         "bucket",
		MIME:           "image/jpeg",
		Size:           100,
		FileName:       "invalid.jpg",
	})
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// Create with nonexistent message -> ErrMessageNotFound
	invalidMsgID := int64(999999)
	_, err = f.repo.CreateAttachment(f.ctxUser1, repository.CreateAttachmentParams{
		ConversationID: conv.ID,
		MessageID:      &invalidMsgID,
		ObjectKey:      "invalid.jpg",
		Bucket:         "bucket",
		MIME:           "image/jpeg",
		Size:           100,
		FileName:       "invalid.jpg",
	})
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)

	// User isolation: User 2 cannot list attachments for User 1's conversation
	_, err = f.repo.ListAttachmentsByConversation(f.ctxUser2, conv.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)

	// User isolation: User 2 cannot list attachments for User 1's message
	_, err = f.repo.ListAttachmentsByMessage(f.ctxUser2, msg.ID)
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)
}

func TestChat_ContextValidation(t *testing.T) {
	f := setupChatRepo(t)

	// No workspace context
	noWsCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{ID: f.user1.ID})

	_, err := f.repo.CreateConversation(noWsCtx, "Test")
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, _, err = f.repo.ListConversations(noWsCtx, 1, 10)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.GetConversation(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, _, err = f.repo.GetConversationWithMessages(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	err = f.repo.DeleteConversation(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.AppendMessage(noWsCtx, repository.AppendMessageParams{})
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	err = f.repo.UpdateConversationTitle(noWsCtx, 1, "Title")
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.ListMessages(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.UpdateDraftStatus(noWsCtx, 1, agentchatmessage.DraftStatusConfirmed, nil)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.CreateAttachment(noWsCtx, repository.CreateAttachmentParams{})
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.ListAttachmentsByConversation(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	_, err = f.repo.ListAttachmentsByMessage(noWsCtx, 1)
	assert.ErrorIs(t, err, base.ErrWorkspaceRequired)

	// No user context (unauthenticated)
	noUserCtx := base.WithWorkspaceID(context.Background(), f.ws1ID)

	_, err = f.repo.CreateConversation(noUserCtx, "Test")
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, _, err = f.repo.ListConversations(noUserCtx, 1, 10)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.GetConversation(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, _, err = f.repo.GetConversationWithMessages(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	err = f.repo.UpdateConversationTitle(noUserCtx, 1, "Title")
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	err = f.repo.DeleteConversation(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.AppendMessage(noUserCtx, repository.AppendMessageParams{})
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.ListMessages(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.UpdateDraftStatus(noUserCtx, 1, agentchatmessage.DraftStatusConfirmed, nil)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.CreateAttachment(noUserCtx, repository.CreateAttachmentParams{})
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.ListAttachmentsByConversation(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)

	_, err = f.repo.ListAttachmentsByMessage(noUserCtx, 1)
	assert.ErrorIs(t, err, base.ErrUnauthorized)
}

func TestChatRepository_GetMessageAndAttachment(t *testing.T) {
	f := setupChatRepo(t)

	conv, err := f.repo.CreateConversation(f.ctxUser1, "Test Message and Attachment")
	require.NoError(t, err)

	msg, err := f.repo.AppendMessage(f.ctxUser1, repository.AppendMessageParams{
		ConversationID: conv.ID,
		Role:           agentchatmessage.RoleUser,
		Content:        "Hello world",
	})
	require.NoError(t, err)

	att, err := f.repo.CreateAttachment(f.ctxUser1, repository.CreateAttachmentParams{
		ConversationID: conv.ID,
		MessageID:      &msg.ID,
		ObjectKey:      "chat/test.pdf",
		Bucket:         "omed-bucket",
		MIME:           "application/pdf",
		Size:           1234,
		FileName:       "test.pdf",
	})
	require.NoError(t, err)

	// GetMessage success
	gotMsg, err := f.repo.GetMessage(f.ctxUser1, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, msg.ID, gotMsg.ID)
	assert.Equal(t, "Hello world", gotMsg.Content)

	// GetMessage isolation (user2 cannot see user1 message)
	_, err = f.repo.GetMessage(f.ctxUser2, msg.ID)
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)

	// GetMessage not found
	_, err = f.repo.GetMessage(f.ctxUser1, 999999)
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)

	// GetAttachment success
	gotAtt, err := f.repo.GetAttachment(f.ctxUser1, att.ID)
	require.NoError(t, err)
	assert.Equal(t, att.ID, gotAtt.ID)
	assert.Equal(t, "test.pdf", gotAtt.FileName)

	// GetAttachment isolation (user2 cannot see user1 attachment)
	_, err = f.repo.GetAttachment(f.ctxUser2, att.ID)
	assert.ErrorIs(t, err, domain.ErrAttachmentNotFound)

	// GetAttachment not found
	_, err = f.repo.GetAttachment(f.ctxUser1, 999999)
	assert.ErrorIs(t, err, domain.ErrAttachmentNotFound)
}
