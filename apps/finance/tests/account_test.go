package tests

import (
	"fmt"
	"strings"
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

// NOTE: jalankan serial (`go test ./tests/... -p 1`) karena JWKS mock bind ke port tetap.

var codeSeq atomic.Int64

func nextCode() string {
	return fmt.Sprintf("T%06d", codeSeq.Add(1))
}

func newAccountReq(t model.AccountType) model.CreateAccountRequest {
	return model.CreateAccountRequest{
		Code:     nextCode(),
		Name:     "Kas Kecil",
		Type:     t,
		Currency: "IDR",
	}
}

type AccountTestSuite struct {
	testutil.ApiTestSuite[model.Account]
}

func (s *AccountTestSuite) SetupSuite() {
	state := testutil.GetState()

	entryRepo := repository.NewEntryRepository(state.EntClient, state.Log)
	repo := repository.NewAccountRepository(state.EntClient, state.Log)
	svc := service.NewAccountService(repo, entryRepo, state.Log)
	controller.NewAccountController(svc).Register(state.Api)

	lpRepo := repository.NewLedgerPeriodRepository(state.EntClient, state.Log)
	lpSvc := service.NewLedgerPeriodService(lpRepo, state.Log)
	controller.NewLedgerPeriodController(lpSvc).Register(state.Api)

	entrySvc := service.NewEntryService(entryRepo, lpRepo, repo, state.Log)
	controller.NewEntryController(entrySvc).Register(state.Api)
}

// ---------------------------------------------------------------- helpers

func (s *AccountTestSuite) withRoles(roles ...shared.WorkspaceRole) {
	s.User.WorkspaceRoles = roles
}

// create membuat account lewat API sebagai user saat ini (harus punya write permission).
func (s *AccountTestSuite) create(req model.CreateAccountRequest) model.Account {
	s.T().Helper()
	s.Request("/accounts", fiber.MethodPost, req)
	s.Created()
	return s.GetResponse().Data
}

func accountPath(id uuid.UUID) string {
	return "/accounts/" + id.String()
}

// ---------------------------------------------------------------- create

func (s *AccountTestSuite) TestCreate() {
	desc := "Petty cash kantor"
	req := newAccountReq(model.AccountTypeAsset)
	req.Description = &desc

	got := s.create(req)

	s.NotEqual(uuid.Nil, got.ID)
	s.Equal(s.User.WorkspaceID, got.WorkspaceID)
	s.Equal(req.Code, got.Code)
	s.Equal(req.Name, got.Name)
	s.Equal(&desc, got.Description)
	s.Equal(model.AccountTypeAsset, got.Type)
	s.Equal("IDR", got.Currency)
	s.Equal(model.AccountStatusActive, got.Status)
	s.Nil(got.ParentID)
	s.Equal(s.User.ID, got.CreatedBy)
	s.Equal(s.User.Name, got.CreatedByName)
	s.Equal(s.User.ID, got.UpdatedBy)
}

func (s *AccountTestSuite) TestCreateWithParent() {
	parent := s.create(newAccountReq(model.AccountTypeAsset))

	req := newAccountReq(model.AccountTypeAsset)
	req.ParentID = &parent.ID
	child := s.create(req)

	s.Require().NotNil(child.ParentID)
	s.Equal(parent.ID, *child.ParentID)
}

func (s *AccountTestSuite) TestCreateValidation() {
	cases := map[string]func(r *model.CreateAccountRequest){
		"code kosong":        func(r *model.CreateAccountRequest) { r.Code = "" },
		"code > 32 char":     func(r *model.CreateAccountRequest) { r.Code = strings.Repeat("x", 33) },
		"name kosong":        func(r *model.CreateAccountRequest) { r.Name = "" },
		"name > 255 char":    func(r *model.CreateAccountRequest) { r.Name = strings.Repeat("x", 256) },
		"type tidak valid":   func(r *model.CreateAccountRequest) { r.Type = "bogus" },
		"currency kosong":    func(r *model.CreateAccountRequest) { r.Currency = "" },
		"currency 2 huruf":   func(r *model.CreateAccountRequest) { r.Currency = "ID" },
		"currency lowercase": func(r *model.CreateAccountRequest) { r.Currency = "idr" },
		"description > 1000": func(r *model.CreateAccountRequest) {
			d := strings.Repeat("x", 1001)
			r.Description = &d
		},
	}

	for name, mutate := range cases {
		s.Run(name, func() {
			req := newAccountReq(model.AccountTypeAsset)
			mutate(&req)
			s.Request("/accounts", fiber.MethodPost, req)
			s.AssertStatus(fiber.StatusUnprocessableEntity)
		})
	}
}

// ---------------------------------------------------------------- list

func (s *AccountTestSuite) TestListEmpty() {
	s.Request("/accounts", fiber.MethodGet, nil)
	s.OK()
	s.Empty(s.PagedResponse().Data)
}

func (s *AccountTestSuite) TestListAndFilter() {
	asset := s.create(newAccountReq(model.AccountTypeAsset))
	expense := s.create(newAccountReq(model.AccountTypeExpense))

	// archive salah satu supaya filter status bisa dites
	archived := model.AccountStatusArchived
	s.Request(accountPath(expense.ID), fiber.MethodPut, model.UpdateAccountRequest{Status: &archived})
	s.OK()

	s.Run("tanpa filter", func() {
		s.Request("/accounts", fiber.MethodGet, nil)
		s.OK()
		s.Len(s.PagedResponse().Data, 2)
	})

	s.Run("filter type", func() {
		s.Request("/accounts?type=asset", fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(asset.ID, data[0].ID)
	})

	s.Run("filter status", func() {
		s.Request("/accounts?status=archived", fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(expense.ID, data[0].ID)
	})

	s.Run("filter tidak valid", func() {
		s.Request("/accounts?type=bogus", fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})
}

// ---------------------------------------------------------------- get

func (s *AccountTestSuite) TestGet() {
	created := s.create(newAccountReq(model.AccountTypeRevenue))

	s.Request(accountPath(created.ID), fiber.MethodGet, nil)
	s.OK()

	got := s.GetResponse().Data
	s.Equal(created.ID, got.ID)
	s.Equal(created.Code, got.Code)
	s.Equal(model.AccountTypeRevenue, got.Type)
}

func (s *AccountTestSuite) TestGetNotFound() {
	s.Request(accountPath(shared.GenerateID()), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountTestSuite) TestGetInvalidID() {
	s.Request("/accounts/not-a-uuid", fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)
}

// ---------------------------------------------------------------- update

func (s *AccountTestSuite) TestUpdate() {
	created := s.create(newAccountReq(model.AccountTypeAsset))

	name := "Kas Besar"
	desc := "diperbarui"
	status := model.AccountStatusArchived
	s.Request(accountPath(created.ID), fiber.MethodPut, model.UpdateAccountRequest{
		Name:        &name,
		Description: &desc,
		Status:      &status,
	})
	s.OK()

	got := s.GetResponse().Data
	s.Equal(name, got.Name)
	s.Equal(&desc, got.Description)
	s.Equal(model.AccountStatusArchived, got.Status)
	s.Equal(s.User.ID, got.UpdatedBy)

	// field immutable tidak berubah
	s.Equal(created.Code, got.Code)
	s.Equal(created.Type, got.Type)
	s.Equal(created.Currency, got.Currency)
}

func (s *AccountTestSuite) TestUpdateIgnoresImmutableFields() {
	created := s.create(newAccountReq(model.AccountTypeAsset))

	s.Request(accountPath(created.ID), fiber.MethodPut, map[string]any{
		"name":     "Baru",
		"code":     "HACKED",
		"type":     "expense",
		"currency": "USD",
	})
	s.OK()

	got := s.GetResponse().Data
	s.Equal("Baru", got.Name)
	s.Equal(created.Code, got.Code)
	s.Equal(model.AccountTypeAsset, got.Type)
	s.Equal("IDR", got.Currency)
}

func (s *AccountTestSuite) TestUpdateValidation() {
	created := s.create(newAccountReq(model.AccountTypeAsset))

	bad := model.AccountStatus("bogus")
	s.Request(accountPath(created.ID), fiber.MethodPut, model.UpdateAccountRequest{Status: &bad})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *AccountTestSuite) TestUpdateNotFound() {
	name := "x"
	s.Request(accountPath(shared.GenerateID()), fiber.MethodPut, model.UpdateAccountRequest{Name: &name})
	s.AssertStatus(fiber.StatusNotFound)
}

// ---------------------------------------------------------------- delete

func (s *AccountTestSuite) TestDelete() {
	created := s.create(newAccountReq(model.AccountTypeAsset))

	s.Request(accountPath(created.ID), fiber.MethodDelete, nil)
	s.NoContent()

	s.Request(accountPath(created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountTestSuite) TestDeleteNotFound() {
	s.Request(accountPath(shared.GenerateID()), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

// ---------------------------------------------------------------- RBAC
// Policy: owner=*, admin=accounts r/w, member=accounts read, user=users read saja.

func (s *AccountTestSuite) TestRBAC() {
	type op struct {
		name   string
		method string
		path   func(id uuid.UUID) string
		body   func() any
		write  bool
	}
	ops := []op{
		{"list", fiber.MethodGet, func(uuid.UUID) string { return "/accounts" }, func() any { return nil }, false},
		{"get", fiber.MethodGet, accountPath, func() any { return nil }, false},
		{"create", fiber.MethodPost, func(uuid.UUID) string { return "/accounts" },
			func() any { return newAccountReq(model.AccountTypeAsset) }, true},
		{"update", fiber.MethodPut, accountPath,
			func() any { n := "rbac"; return model.UpdateAccountRequest{Name: &n} }, true},
		{"delete", fiber.MethodDelete, accountPath, func() any { return nil }, true},
	}

	successStatus := map[string]int{
		"list":   fiber.StatusOK,
		"get":    fiber.StatusOK,
		"create": fiber.StatusCreated,
		"update": fiber.StatusOK,
		"delete": fiber.StatusNoContent,
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

	for _, r := range roles {
		for _, o := range ops {
			s.Run(r.name+"/"+o.name, func() {
				// seed sebagai owner supaya selalu ada target
				s.withRoles(shared.WorkspaceRoleOwner)
				seed := s.create(newAccountReq(model.AccountTypeAsset))

				s.withRoles(r.roles...)
				s.Request(o.path(seed.ID), o.method, o.body())

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

// ---------------------------------------------------------------- workspace isolation

func (s *AccountTestSuite) TestWorkspaceIsolation() {
	wsA := s.User.WorkspaceID
	created := s.create(newAccountReq(model.AccountTypeAsset))

	// pindah ke workspace lain (user sama, role owner)
	s.User.WorkspaceID = shared.GenerateID()
	s.User.WorkspaceName = "Other Workspace"

	s.Run("get tidak terlihat", func() {
		s.Request(accountPath(created.ID), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("list kosong", func() {
		s.Request("/accounts", fiber.MethodGet, nil)
		s.OK()
		s.Empty(s.PagedResponse().Data)
	})

	s.Run("update ditolak", func() {
		name := "hijack"
		s.Request(accountPath(created.ID), fiber.MethodPut, model.UpdateAccountRequest{Name: &name})
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("delete ditolak", func() {
		s.Request(accountPath(created.ID), fiber.MethodDelete, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("code yang sama boleh di workspace lain", func() {
		req := model.CreateAccountRequest{
			Code:     created.Code,
			Name:     "Sama tapi beda workspace",
			Type:     model.AccountTypeAsset,
			Currency: "IDR",
		}
		other := s.create(req)
		s.NotEqual(created.ID, other.ID)
		s.Equal(s.User.WorkspaceID, other.WorkspaceID)
	})

	// balik ke workspace A: data asli masih utuh
	s.User.WorkspaceID = wsA
	s.Request(accountPath(created.ID), fiber.MethodGet, nil)
	s.OK()
	got := s.GetResponse().Data
	s.Equal(created.Name, got.Name)
	s.Equal(wsA, got.WorkspaceID)
}

// ---------------------------------------------------------------- seed

func (s *AccountTestSuite) TestSeed() {
	s.Run("success seed freelancer en", func() {
		s.Request("/accounts/seed", fiber.MethodPost, model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
		})
		s.Created()

		data := s.PagedResponse().Data
		s.NotEmpty(data)

		// Verifikasi akun hirarki terbentuk di database
		s.Request("/accounts", fiber.MethodGet, nil)
		s.OK()
		allAccounts := s.PagedResponse().Data
		s.NotEmpty(allAccounts)

		// Pastikan ada akun 1000 (parent) dan 1100 (child) dengan parentID yang valid
		var parent, child *model.Account
		for i := range allAccounts {
			acc := &allAccounts[i]
			if acc.Code == "1000" {
				parent = acc
			}
			if acc.Code == "1100" {
				child = acc
			}
		}

		s.Require().NotNil(parent, "Account parent 1000 harus terbuat")
		s.Require().NotNil(child, "Account child 1100 harus terbuat")
		s.Require().NotNil(child.ParentID, "ParentID child 1100 harus terisi")
		s.Equal(parent.ID, *child.ParentID, "ParentID child 1100 harus menunjuk ke ID parent 1000")
	})

	s.Run("cannot seed when accounts > 0 and force is false", func() {
		// Workspace ini sudah memiliki akun dari subtest sebelumnya
		s.Request("/accounts/seed", fiber.MethodPost, model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
			Force:   false,
		})
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})

	s.Run("cannot seed when entries > 0 even with force true", func() {
		// Dapatkan akun aktif dari workspace
		s.Request("/accounts", fiber.MethodGet, nil)
		s.OK()
		accs := s.PagedResponse().Data
		s.Require().True(len(accs) >= 2)

		now := time.Now().Truncate(24 * time.Hour)
		// Buat periode aktif
		s.Request("/ledger-periods", fiber.MethodPost, model.CreateLedgerPeriodRequest{
			StartDate: now.AddDate(0, -1, 0),
			EndDate:   now.AddDate(0, 1, 0),
		})
		s.Created()

		// Buat entri jurnal
		s.Request("/entries", fiber.MethodPost, model.CreateEntryRequest{
			EntryDate:   now,
			Description: "Test entry for seed block",
			Postings: []model.CreatePostingRequest{
				{AccountID: accs[0].ID, Currency: "IDR", DebitAmount: decimal.NewFromInt(1000), CreditAmount: decimal.Zero},
				{AccountID: accs[1].ID, Currency: "IDR", DebitAmount: decimal.Zero, CreditAmount: decimal.NewFromInt(1000)},
			},
		})
		s.Created()

		// Coba force seed -> harus ditolak karena ada entri jurnal
		s.Request("/accounts/seed", fiber.MethodPost, model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
			Force:   true,
		})
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})

	s.Run("success force seed when accounts > 0 and entries == 0", func() {
		// Workspace baru
		s.User.WorkspaceID = shared.GenerateID()
		s.User.WorkspaceName = "Fresh Seed Workspace"

		// Buat akun manual
		s.create(newAccountReq(model.AccountTypeAsset))

		// Force seed
		s.Request("/accounts/seed", fiber.MethodPost, model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
			Force:   true,
		})
		s.Created()

		s.Request("/accounts", fiber.MethodGet, nil)
		s.OK()
		allAccounts := s.PagedResponse().Data
		s.NotEmpty(allAccounts)
	})

	s.Run("invalid profile or lang", func() {
		s.User.WorkspaceID = shared.GenerateID()
		s.Request("/accounts/seed", fiber.MethodPost, model.SeedAccountRequest{
			Profile: "nonexistent",
			Lang:    "en",
		})
		s.AssertStatus(fiber.StatusInternalServerError)
	})
}

func TestAccountSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}
