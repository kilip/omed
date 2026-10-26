// Package testctl provides development and test-only HTTP endpoints to support E2E tests.
package testctl

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"runtime/coverage"
	"strconv"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/file"
	"github.com/kilip/omed/api/ent/session"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/ent/workspaceinvitation"
	"github.com/kilip/omed/api/ent/workspacemember"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/mailx"
)

// Controller handles development and test-only helper endpoints.
type Controller struct {
	client     *ent.Client
	mailer     mailx.Mailer
	cookieName string
}

// NewController creates a new test Controller instance.
func NewController(client *ent.Client, mailer mailx.Mailer, cookieName ...string) Controller {
	cn := "token"
	if len(cookieName) > 0 && cookieName[0] != "" {
		cn = cookieName[0]
	}
	return Controller{
		client:     client,
		mailer:     mailer,
		cookieName: cn,
	}
}

// RegisterRoutes mounts all test utility routes on the provided router.
func (ctl Controller) RegisterRoutes(r fiber.Router) {
	test := r.Group("/test")
	test.Post("/users", ctl.EnsureUser)
	test.Post("/sessions", ctl.CreateSession)
	test.Post("/workspaces", ctl.CreateWorkspace)
	test.Post("/members", ctl.AddMember)
	test.Delete("/users/:id", ctl.DeleteUser)
	test.Post("/reset", ctl.Reset)
	test.Post("/coverage/flush", ctl.FlushCoverage)

	test.Get("/mailer/messages", ctl.GetMailerMessages)
	test.Get("/mailer/last", ctl.GetMailerLastMessage)
	test.Post("/mailer/reset", ctl.ResetMailer)
}

func harnessCtx(ctx context.Context, userID ...int64) context.Context {
	uid := int64(1)
	if len(userID) > 0 && userID[0] > 0 {
		uid = userID[0]
	}
	return context.WithValue(ctx, base.USER_CONTEXT_KEY, base.UserContext{
		ID:            uid,
		Name:          "Test Harness",
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})
}

type ensureUserRequest struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

type userResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar,omitempty"`
}

// EnsureUser creates or updates a user by email and returns the user details.
func (ctl Controller) EnsureUser(c fiber.Ctx) error {
	var req ensureUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}
	if req.Name == "" {
		req.Name = "Test User"
	}

	ctx := harnessCtx(c.Context())

	b := ctl.client.User.Create().
		SetName(req.Name).
		SetEmail(req.Email)
	if req.Avatar != "" {
		b.SetAvatar(req.Avatar)
	}

	id, err := b.OnConflict(
		sql.ConflictColumns(user.FieldEmail),
	).UpdateNewValues().ID(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to ensure user: "+err.Error())
	}

	usr, err := ctl.client.User.Get(ctx, id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get user: "+err.Error())
	}

	return c.JSON(userResponse{
		ID:     usr.ID,
		Name:   usr.Name,
		Email:  usr.Email,
		Avatar: usr.Avatar,
	})
}

type createSessionRequest struct {
	UserID int64 `json:"userId"`
}

type sessionResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Cookie    string    `json:"cookie"`
}

// CreateSession creates an active session for the specified user and sets the session cookie.
func (ctl Controller) CreateSession(c fiber.Ctx) error {
	var req createSessionRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.UserID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "userId must be positive")
	}

	ctx := harnessCtx(c.Context(), req.UserID)

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}
	token := hex.EncodeToString(bytes)
	expiresAt := time.Now().Add(24 * time.Hour)

	h := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(h[:])

	_, err := ctl.client.Session.Create().
		SetUserID(req.UserID).
		SetToken(tokenHash).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create session: "+err.Error())
	}

	c.Cookie(&fiber.Cookie{
		Name:     ctl.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.JSON(sessionResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Cookie:    ctl.cookieName + "=" + token + "; Path=/; HttpOnly; SameSite=Lax",
	})
}

type createWorkspaceRequest struct {
	Name     string `json:"name"`
	OwnerID  int64  `json:"ownerId"`
	Personal bool   `json:"personal"`
}

type workspaceResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	OwnerID  int64  `json:"ownerId"`
	Personal bool   `json:"personal"`
}

// CreateWorkspace provisions a workspace for a test user and adds them as an owner member.
func (ctl Controller) CreateWorkspace(c fiber.Ctx) error {
	var req createWorkspaceRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.OwnerID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "ownerId must be positive")
	}
	if req.Name == "" {
		req.Name = "Test Workspace"
	}

	ctx := harnessCtx(c.Context(), req.OwnerID)

	w, err := ctl.client.Workspace.Create().
		SetName(req.Name).
		SetOwnerID(req.OwnerID).
		SetPersonal(req.Personal).
		Save(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create workspace: "+err.Error())
	}

	// Add membership
	_, err = ctl.client.WorkspaceMember.Create().
		SetWorkspaceID(w.ID).
		SetUserID(req.OwnerID).
		SetRoles([]string{"owner"}).
		Save(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to add member: "+err.Error())
	}

	return c.JSON(workspaceResponse{
		ID:       w.ID,
		Name:     w.Name,
		OwnerID:  w.OwnerID,
		Personal: w.Personal,
	})
}

type addMemberRequest struct {
	WorkspaceID int64    `json:"workspaceId"`
	UserID      int64    `json:"userId"`
	Roles       []string `json:"roles"`
	Role        string   `json:"role"`
}

// AddMember adds or updates workspace membership for a user.
func (ctl Controller) AddMember(c fiber.Ctx) error {
	var req addMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.WorkspaceID <= 0 || req.UserID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "workspaceId and userId are required")
	}
	if len(req.Roles) == 0 && req.Role != "" {
		req.Roles = []string{req.Role}
	}
	if len(req.Roles) == 0 {
		req.Roles = []string{"member"}
	}

	actorID := req.UserID
	if ws, err := ctl.client.Workspace.Get(c.Context(), req.WorkspaceID); err == nil && ws.OwnerID > 0 {
		actorID = ws.OwnerID
	}
	ctx := harnessCtx(c.Context(), actorID)

	err := ctl.client.WorkspaceMember.Create().
		SetWorkspaceID(req.WorkspaceID).
		SetUserID(req.UserID).
		SetRoles(req.Roles).
		OnConflict(
			sql.ConflictColumns(workspacemember.FieldWorkspaceID, workspacemember.FieldUserID),
		).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to add member: "+err.Error())
	}

	return c.JSON(fiber.Map{"success": true})
}

// DeleteUser deletes a user and associated sessions and memberships.
func (ctl Controller) DeleteUser(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid user id")
	}

	ctx := harnessCtx(c.Context(), id)

	wsIDs, _ := ctl.client.Workspace.Query().Where(workspace.OwnerID(id)).IDs(ctx)
	if len(wsIDs) > 0 {
		unfiltered := base.WithSkipWorkspaceFilter(ctx)
		_, _ = ctl.client.File.Delete().Where(file.WorkspaceIDIn(wsIDs...)).Exec(unfiltered)
		_, _ = ctl.client.WorkspaceInvitation.Delete().Where(workspaceinvitation.WorkspaceIDIn(wsIDs...)).Exec(ctx)
		_, _ = ctl.client.WorkspaceMember.Delete().Where(workspacemember.WorkspaceIDIn(wsIDs...)).Exec(ctx)
	}
	_, _ = ctl.client.File.Delete().Where(file.Or(
		file.CreatedBy(id),
		file.UpdatedBy(id),
	)).Exec(base.WithSkipWorkspaceFilter(ctx))
	_, _ = ctl.client.WorkspaceInvitation.Delete().Where(workspaceinvitation.Or(
		workspaceinvitation.CreatedBy(id),
		workspaceinvitation.UpdatedBy(id),
	)).Exec(ctx)
	_, _ = ctl.client.Session.Delete().Where(session.UserID(id)).Exec(ctx)
	_, _ = ctl.client.WorkspaceMember.Delete().Where(workspacemember.UserID(id)).Exec(ctx)
	_, _ = ctl.client.Workspace.Delete().Where(workspace.OwnerID(id)).Exec(ctx)
	_, _ = ctl.client.User.Delete().Where(user.ID(id)).Exec(ctx)

	return c.JSON(fiber.Map{"success": true})
}

// Reset cleans up all test users and associated test data.
func (ctl Controller) Reset(c fiber.Ctx) error {
	ctx := harnessCtx(c.Context())

	testUsers, err := ctl.client.User.Query().
		Where(user.EmailHasSuffix("@omed.test")).
		All(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to query test users: "+err.Error())
	}

	for _, u := range testUsers {
		uCtx := harnessCtx(c.Context(), u.ID)
		wsIDs, _ := ctl.client.Workspace.Query().Where(workspace.OwnerID(u.ID)).IDs(uCtx)
		if len(wsIDs) > 0 {
			unfiltered := base.WithSkipWorkspaceFilter(uCtx)
			_, _ = ctl.client.File.Delete().Where(file.WorkspaceIDIn(wsIDs...)).Exec(unfiltered)
			_, _ = ctl.client.WorkspaceInvitation.Delete().Where(workspaceinvitation.WorkspaceIDIn(wsIDs...)).Exec(uCtx)
			_, _ = ctl.client.WorkspaceMember.Delete().Where(workspacemember.WorkspaceIDIn(wsIDs...)).Exec(uCtx)
		}
		_, _ = ctl.client.File.Delete().Where(file.Or(
			file.CreatedBy(u.ID),
			file.UpdatedBy(u.ID),
		)).Exec(base.WithSkipWorkspaceFilter(uCtx))
		_, _ = ctl.client.WorkspaceInvitation.Delete().Where(workspaceinvitation.Or(
			workspaceinvitation.CreatedBy(u.ID),
			workspaceinvitation.UpdatedBy(u.ID),
		)).Exec(uCtx)
		_, _ = ctl.client.Session.Delete().Where(session.UserID(u.ID)).Exec(uCtx)
		_, _ = ctl.client.WorkspaceMember.Delete().Where(workspacemember.UserID(u.ID)).Exec(uCtx)
		_, _ = ctl.client.Workspace.Delete().Where(workspace.OwnerID(u.ID)).Exec(uCtx)
		_, _ = ctl.client.User.Delete().Where(user.ID(u.ID)).Exec(uCtx)
	}

	if md := ctl.getMemoryDriver(); md != nil {
		md.Reset()
	}

	return c.JSON(fiber.Map{"success": true, "deletedCount": len(testUsers)})
}

// FlushCoverage writes runtime coverage data to GOCOVERDIR if set.
func (ctl Controller) FlushCoverage(c fiber.Ctx) error {
	dir := os.Getenv("GOCOVERDIR")
	if dir != "" {
		_ = coverage.WriteMetaDir(dir)
		_ = coverage.WriteCountersDir(dir)
	}
	return c.JSON(fiber.Map{"success": true, "dir": dir})
}

func (ctl Controller) getMemoryDriver() *mailx.MemoryDriver {
	if ctl.mailer == nil {
		return nil
	}
	md, _ := ctl.mailer.Driver().(*mailx.MemoryDriver)
	return md
}

// GetMailerMessages returns all messages dispatched through the in-memory driver.
func (ctl Controller) GetMailerMessages(c fiber.Ctx) error {
	md := ctl.getMemoryDriver()
	if md == nil {
		return c.JSON([]mailx.Message{})
	}
	msgs := md.SentMessages()
	if msgs == nil {
		msgs = []mailx.Message{}
	}
	return c.JSON(msgs)
}

// GetMailerLastMessage returns the most recently dispatched message.
func (ctl Controller) GetMailerLastMessage(c fiber.Ctx) error {
	md := ctl.getMemoryDriver()
	if md == nil {
		return fiber.NewError(fiber.StatusNotFound, "no messages found")
	}
	msg := md.LastMessage()
	if msg == nil {
		return fiber.NewError(fiber.StatusNotFound, "no messages found")
	}
	return c.JSON(msg)
}

// ResetMailer clears all messages recorded in the in-memory driver.
func (ctl Controller) ResetMailer(c fiber.Ctx) error {
	if md := ctl.getMemoryDriver(); md != nil {
		md.Reset()
	}
	return c.JSON(fiber.Map{"success": true})
}
