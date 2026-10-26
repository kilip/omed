package seed_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SeederSuite struct {
	suite.Suite
	client      *ent.Client
	workspaceID int64
	userID      int64
}

func TestSeederSuite(t *testing.T) {
	suite.Run(t, new(SeederSuite))
}

func (s *SeederSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:seeder_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
}

func (s *SeederSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *SeederSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Freelancer Toni").
		SetEmail("toni@freelance.id").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Freelancer Toni",
	})
	w, err := s.client.Workspace.Create().
		SetName("Freelance Workspace").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *SeederSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Freelancer Toni",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *SeederSuite) Test_LoadTemplate() {
	// Valid ID template
	itemsID, err := seed.LoadTemplate("freelancer", "id")
	s.Require().NoError(err)
	s.Greater(len(itemsID), 40)

	// Check specific Indonesian banks & ewallets exist
	foundBCA := false
	foundGoPay := false
	foundPrive := false
	foundPayLater := false
	foundMakan := false
	for _, item := range itemsID {
		if item.Code == "1121" && item.Name == "Bank BCA" {
			foundBCA = true
		}
		if item.Code == "1112" && item.Name == "GoPay" {
			foundGoPay = true
		}
		if item.Code == "3200" && item.Name == "Prive Pemilik" {
			foundPrive = true
		}
		if item.Code == "2160" && item.Name == "Utang PayLater & Pinjaman Singkat" {
			foundPayLater = true
		}
		if item.Code == "6710" && item.Name == "Makanan & Minuman" {
			foundMakan = true
		}
	}
	s.True(foundBCA, "Bank BCA should be in freelancer ID template")
	s.True(foundGoPay, "GoPay should be in freelancer ID template")
	s.True(foundPrive, "Prive Pemilik should be in freelancer ID template")
	s.True(foundPayLater, "PayLater should be in freelancer ID template")
	s.True(foundMakan, "Makanan & Minuman should be in freelancer ID template")

	// Valid EN template
	itemsEN, err := seed.LoadTemplate("freelancer", "en")
	s.Require().NoError(err)
	s.Greater(len(itemsEN), 30)

	// Invalid template
	_, err = seed.LoadTemplate("nonexistent", "id")
	s.Require().ErrorIs(err, domain.ErrAccountTemplateNotFound)

	// Invalid language
	_, err = seed.LoadTemplate("freelancer", "jp")
	s.Require().ErrorIs(err, domain.ErrAccountTemplateNotFound)
}

func (s *SeederSuite) Test_Seed_SuccessAndHierarchy() {
	ctx := s.ctx()

	count, err := seed.Seed(ctx, s.client, s.workspaceID, "freelancer", "id")
	s.Require().NoError(err)
	s.Greater(count, 40)

	// Verify all accounts exist in workspace
	accounts, err := s.client.Account.Query().All(ctx)
	s.Require().NoError(err)
	s.Len(accounts, count)

	// Verify parent child link: 1111 (Kas Tunai) parent should be 1110 (Kas & E-Wallet)
	var kasTunai, kasGroup *ent.Account
	for _, a := range accounts {
		if a.Code == "1111" {
			kasTunai = a
		}
		if a.Code == "1110" {
			kasGroup = a
		}
	}
	s.Require().NotNil(kasTunai)
	s.Require().NotNil(kasGroup)
	s.Require().NotNil(kasTunai.ParentID)
	s.Equal(kasGroup.ID, *kasTunai.ParentID)
	s.True(kasTunai.IsPostable)
	s.False(kasGroup.IsPostable)

	// Verify FX accounts
	accByCode := make(map[string]*ent.Account, len(accounts))
	for _, a := range accounts {
		accByCode[a.Code] = a
	}
	s.Require().NotNil(accByCode["7200"])
	s.False(accByCode["7200"].IsPostable)
	s.Require().NotNil(accByCode["7210"])
	s.True(accByCode["7210"].IsPostable)
	s.Equal(accByCode["7200"].ID, *accByCode["7210"].ParentID)
	s.Require().NotNil(accByCode["7220"])
	s.True(accByCode["7220"].IsPostable)
	s.Equal(accByCode["7200"].ID, *accByCode["7220"].ParentID)

	s.Require().NotNil(accByCode["7300"])
	s.False(accByCode["7300"].IsPostable)
	s.Require().NotNil(accByCode["7310"])
	s.True(accByCode["7310"].IsPostable)
	s.Equal(accByCode["7300"].ID, *accByCode["7310"].ParentID)
	s.Require().NotNil(accByCode["7320"])
	s.True(accByCode["7320"].IsPostable)
	s.Equal(accByCode["7300"].ID, *accByCode["7320"].ParentID)

	s.Require().NotNil(accByCode["7400"])
	s.True(accByCode["7400"].IsPostable)
	s.Equal(accByCode["7000"].ID, *accByCode["7400"].ParentID)

	// Seeding a second time must fail with ErrAccountWorkspaceNotEmpty
	_, err = seed.Seed(ctx, s.client, s.workspaceID, "freelancer", "id")
	s.Require().ErrorIs(err, domain.ErrAccountWorkspaceNotEmpty)
}

func (s *SeederSuite) Test_Seed_EnglishTemplate() {
	ctx := s.ctx()

	count, err := seed.Seed(ctx, s.client, s.workspaceID, "freelancer", "en")
	s.Require().NoError(err)
	s.Greater(count, 30)

	accounts, err := s.client.Account.Query().All(ctx)
	s.Require().NoError(err)
	s.Len(accounts, count)
}
