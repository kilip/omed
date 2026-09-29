package tests

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/testutil"
)

var (
	entryAccountSeq atomic.Int64
	entryPeriodSeq  atomic.Int64
)

func nextEntryAccountCode() string {
	return fmt.Sprintf("E%06d", entryAccountSeq.Add(1))
}

func nextEntryPeriodName() string {
	return "Entry Period " + strconv.FormatInt(entryPeriodSeq.Add(1), 10)
}

type EntryTestSuite struct {
	testutil.ApiTestSuite[model.Entry]
}

func (s *EntryTestSuite) SetupSuite() {
	state := testutil.GetState()

	entryRepo := repository.NewEntryRepository(state.EntClient, state.Log)
	accRepo := repository.NewAccountRepository(state.EntClient, state.Log)
	accSvc := service.NewAccountService(accRepo, entryRepo, state.Log)
	controller.NewAccountController(accSvc).Register(state.Api)

	lpRepo := repository.NewLedgerPeriodRepository(state.EntClient, state.Log)
	lpSvc := service.NewLedgerPeriodService(lpRepo, state.Log)
	controller.NewLedgerPeriodController(lpSvc).Register(state.Api)

	entrySvc := service.NewEntryService(entryRepo, lpRepo, accRepo, state.Log)
	controller.NewEntryController(entrySvc).Register(state.Api)
}

// ---------------------------------------------------------------- helpers

func (s *EntryTestSuite) withRoles(roles ...shared.WorkspaceRole) {
	s.User.WorkspaceRoles = roles
}

func (s *EntryTestSuite) createAccount(t model.AccountType) model.Account {
	s.T().Helper()
	s.withRoles(shared.WorkspaceRoleOwner)
	req := model.CreateAccountRequest{
		Code:     nextEntryAccountCode(),
		Name:     "Akun Tes",
		Type:     t,
		Currency: "IDR",
	}
	s.Request("/accounts", fiber.MethodPost, req)
	s.Created()

	return testutil.DecodeOther[model.Account](s).Data
}

func (s *EntryTestSuite) createOpenPeriod(start, end time.Time) model.LedgerPeriod {
	s.T().Helper()
	s.withRoles(shared.WorkspaceRoleOwner)
	req := model.CreateLedgerPeriodRequest{
		StartDate: start,
		EndDate:   end,
	}
	s.Request("/ledger-periods", fiber.MethodPost, req)
	s.Created()

	return testutil.DecodeOther[model.LedgerPeriod](s).Data
}


func (s *EntryTestSuite) createEntry(req model.CreateEntryRequest) model.Entry {
	s.T().Helper()
	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/entries", fiber.MethodPost, req)
	s.Created()
	return s.GetResponse().Data
}

func entryPath(id uuid.UUID) string {
	return "/entries/" + id.String()
}

// ---------------------------------------------------------------- tests

func (s *EntryTestSuite) TestCreateBalanced() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	amount := decimal.NewFromInt(150000)

	req := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Beli Perlengkapan",
		Postings: []model.CreatePostingRequest{
			{
				AccountID:    acc1.ID,
				Currency:     "IDR",
				DebitAmount:  amount,
				CreditAmount: decimal.Zero,
			},
			{
				AccountID:    acc2.ID,
				Currency:     "IDR",
				DebitAmount:  decimal.Zero,
				CreditAmount: amount,
			},
		},
	}

	got := s.createEntry(req)

	s.NotEqual(uuid.Nil, got.ID)
	s.Equal(s.User.WorkspaceID, got.WorkspaceID)
	s.Equal(req.Description, got.Description)
	s.Equal(model.EntryTypeNormal, got.EntryType)
	s.Require().Len(got.Postings, 2)
	s.Equal(acc1.ID, got.Postings[0].AccountID)
	s.True(amount.Equal(got.Postings[0].DebitAmount))
	s.Equal(acc2.ID, got.Postings[1].AccountID)
	s.True(amount.Equal(got.Postings[1].CreditAmount))
}

func (s *EntryTestSuite) TestCreateUnbalanced() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	req := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Jurnal Tidak Seimbang",
		Postings: []model.CreatePostingRequest{
			{
				AccountID:    acc1.ID,
				Currency:     "IDR",
				DebitAmount:  decimal.NewFromInt(100000),
				CreditAmount: decimal.Zero,
			},
			{
				AccountID:    acc2.ID,
				Currency:     "IDR",
				DebitAmount:  decimal.Zero,
				CreditAmount: decimal.NewFromInt(50000), // debit != credit
			},
		},
	}

	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/entries", fiber.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *EntryTestSuite) TestCreateMinPostingsCount() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)

	req := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Hanya 1 Baris Posting",
		Postings: []model.CreatePostingRequest{
			{
				AccountID:    acc1.ID,
				Currency:     "IDR",
				DebitAmount:  decimal.NewFromInt(100000),
				CreditAmount: decimal.Zero,
			},
		},
	}

	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/entries", fiber.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *EntryTestSuite) TestCreateClosedOrMissingPeriod() {
	now := time.Now().Truncate(24 * time.Hour)
	period := s.createOpenPeriod(now.AddDate(0, 0, 10), now.AddDate(0, 1, 0)) // period starts in the future

	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	amount := decimal.NewFromInt(100000)

	s.Run("tanpa periode open pada tanggal jurnal", func() {
		req := model.CreateEntryRequest{
			EntryDate:   now, // before period start
			EntryType:   model.EntryTypeNormal,
			Description: "Tanggal Diluar Periode",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
				{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
			},
		}

		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/entries", fiber.MethodPost, req)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})

	s.Run("pada periode terkunci/tutup", func() {
		// Close the period
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/ledger-periods/"+period.ID.String(), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
			Status: model.LedgerPeriodStatusClosed,
		})
		s.OK()

		req := model.CreateEntryRequest{
			EntryDate:   now.AddDate(0, 0, 15), // inside closed period
			EntryType:   model.EntryTypeNormal,
			Description: "Tanggal Periode Closed",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
				{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
			},
		}

		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/entries", fiber.MethodPost, req)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})
}

func (s *EntryTestSuite) TestCreateArchivedAccount() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	// Archive acc2
	archived := model.AccountStatusArchived
	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/accounts/"+acc2.ID.String(), fiber.MethodPut, model.UpdateAccountRequest{
		Status: &archived,
	})
	s.OK()

	amount := decimal.NewFromInt(100000)
	req := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Posting ke Akun Archived",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
			{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
		},
	}

	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/entries", fiber.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *EntryTestSuite) TestAtomicRollbackOnFailure() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)
	invalidAccID := shared.GenerateID() // Non-existent account

	amount := decimal.NewFromInt(100000)
	req := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Posting ke Akun Tidak Ada",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
			{AccountID: invalidAccID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
		},
	}

	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request("/entries", fiber.MethodPost, req)
	s.AssertStatus(fiber.StatusNotFound)

	// Verify no entry was created
	s.Request("/entries", fiber.MethodGet, nil)
	s.OK()
	s.Empty(s.PagedResponse().Data)
}

func (s *EntryTestSuite) TestGetAndList() {
	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))

	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	amount := decimal.NewFromInt(200000)
	req1 := model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Entry Normal",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
			{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
		},
	}
	entry1 := s.createEntry(req1)

	req2 := model.CreateEntryRequest{
		EntryDate:   now.AddDate(0, 0, 1),
		EntryType:   model.EntryTypeAdjustment,
		Description: "Entry Penyesuaian",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amount, CreditAmount: decimal.Zero},
			{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amount},
		},
	}
	entry2 := s.createEntry(req2)

	s.Run("get by id", func() {
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request(entryPath(entry1.ID), fiber.MethodGet, nil)
		s.OK()
		got := s.GetResponse().Data
		s.Equal(entry1.ID, got.ID)
		s.Equal(entry1.Description, got.Description)
		s.Len(got.Postings, 2)
	})

	s.Run("list without filter", func() {
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/entries", fiber.MethodGet, nil)
		s.OK()
		s.Len(s.PagedResponse().Data, 2)
	})

	s.Run("list filter by entry_type", func() {
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/entries?entry_type=adjustment", fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(entry2.ID, data[0].ID)
	})
}

func (s *EntryTestSuite) TestRBAC() {
	type op struct {
		name   string
		method string
		path   func(id uuid.UUID) string
		body   func(acc1, acc2 uuid.UUID, now time.Time) any
		write  bool
	}
	ops := []op{
		{"list", fiber.MethodGet, func(uuid.UUID) string { return "/entries" }, func(a1, a2 uuid.UUID, t time.Time) any { return nil }, false},
		{"get", fiber.MethodGet, entryPath, func(a1, a2 uuid.UUID, t time.Time) any { return nil }, false},
		{"create", fiber.MethodPost, func(uuid.UUID) string { return "/entries" },
			func(a1, a2 uuid.UUID, t time.Time) any {
				amt := decimal.NewFromInt(100)
				return model.CreateEntryRequest{
					EntryDate:   t,
					EntryType:   model.EntryTypeNormal,
					Description: "RBAC Test Entry",
					Postings: []model.CreatePostingRequest{
						{AccountID: a1, Currency: "IDR", DebitAmount: amt, CreditAmount: decimal.Zero},
						{AccountID: a2, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amt},
					},
				}
			}, true},
	}

	successStatus := map[string]int{
		"list":   fiber.StatusOK,
		"get":    fiber.StatusOK,
		"create": fiber.StatusCreated,
	}

	roles := []struct {
		name  string
		roles []shared.WorkspaceRole
		read  bool
		write bool
	}{
		{"owner", []shared.WorkspaceRole{shared.WorkspaceRoleOwner}, true, true},
		{"admin", []shared.WorkspaceRole{shared.WorkspaceRoleAdmin}, true, true},
		{"member", []shared.WorkspaceRole{shared.WorkspaceRoleMember}, true, false},
		{"user", []shared.WorkspaceRole{shared.WorkspaceRole(shared.UserRoleUser)}, false, false},
		{"tanpa role", nil, false, false},
	}

	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))
	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	for _, r := range roles {
		for _, o := range ops {
			s.Run(r.name+"/"+o.name, func() {
				s.withRoles(shared.WorkspaceRoleOwner)
				amt := decimal.NewFromInt(500)
				seed := s.createEntry(model.CreateEntryRequest{
					EntryDate:   now,
					EntryType:   model.EntryTypeNormal,
					Description: "Seed Entry",
					Postings: []model.CreatePostingRequest{
						{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amt, CreditAmount: decimal.Zero},
						{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amt},
					},
				})

				s.withRoles(r.roles...)
				s.Request(o.path(seed.ID), o.method, o.body(acc1.ID, acc2.ID, now))

				allowed := (o.write && r.write) || (!o.write && r.read)
				if allowed {
					s.AssertStatus(successStatus[o.name])
				} else {
					s.AssertStatus(fiber.StatusForbidden)
				}
			})
		}
	}
}

func (s *EntryTestSuite) TestWorkspaceIsolation() {
	wsA := s.User.WorkspaceID

	now := time.Now().Truncate(24 * time.Hour)
	s.createOpenPeriod(now.AddDate(0, -1, 0), now.AddDate(0, 1, 0))
	acc1 := s.createAccount(model.AccountTypeAsset)
	acc2 := s.createAccount(model.AccountTypeExpense)

	amt := decimal.NewFromInt(1000)
	created := s.createEntry(model.CreateEntryRequest{
		EntryDate:   now,
		EntryType:   model.EntryTypeNormal,
		Description: "Workspace Isolation Entry",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1.ID, Currency: "IDR", DebitAmount: amt, CreditAmount: decimal.Zero},
			{AccountID: acc2.ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: amt},
		},
	})

	// Switch workspace
	s.User.WorkspaceID = shared.GenerateID()
	s.User.WorkspaceName = "Other Workspace"

	s.Run("get tidak terlihat di workspace lain", func() {
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request(entryPath(created.ID), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("list kosong di workspace lain", func() {
		s.withRoles(shared.WorkspaceRoleOwner)
		s.Request("/entries", fiber.MethodGet, nil)
		s.OK()
		s.Empty(s.PagedResponse().Data)
	})

	// Switch back to original workspace
	s.User.WorkspaceID = wsA
	s.withRoles(shared.WorkspaceRoleOwner)
	s.Request(entryPath(created.ID), fiber.MethodGet, nil)
	s.OK()
	got := s.GetResponse().Data
	s.Equal(created.ID, got.ID)
	s.Equal(wsA, got.WorkspaceID)
}

func TestEntrySuite(t *testing.T) {
	suite.Run(t, new(EntryTestSuite))
}
