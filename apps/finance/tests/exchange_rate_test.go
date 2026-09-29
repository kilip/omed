package tests

import (
	"net/url"
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

var exchangeRateSeq atomic.Int64

func newExchangeRateReq() model.CreateExchangeRateRequest {
	seq := exchangeRateSeq.Add(1)
	now := time.Now().Truncate(time.Hour * 24).Add(time.Duration(seq*24) * time.Hour)
	src := "ECB"
	return model.CreateExchangeRateRequest{
		FromCurrency: "USD",
		ToCurrency:   "IDR",
		Rate:         decimal.NewFromFloat(15000 + float64(seq)),
		RateDate:     now,
		Source:       &src,
	}
}

type ExchangeRateTestSuite struct {
	testutil.ApiTestSuite[model.ExchangeRate]
}

func (s *ExchangeRateTestSuite) SetupSuite() {
	state := testutil.GetState()

	repo := repository.NewExchangeRateRepository(state.EntClient, state.Log)
	svc := service.NewExchangeRateService(repo, state.Log)

	controller.NewExchangeRateController(svc).Register(state.Api)
}

func (s *ExchangeRateTestSuite) withRoles(roles ...shared.WorkspaceRole) {
	s.User.WorkspaceRoles = roles
}

func (s *ExchangeRateTestSuite) create(req model.CreateExchangeRateRequest) model.ExchangeRate {
	s.T().Helper()
	s.Request("/exchange-rates", fiber.MethodPost, req)
	s.AssertStatus(fiber.StatusCreated)
	return s.GetResponse().Data
}

func exchangeRatePath(id uuid.UUID) string {
	return "/exchange-rates/" + id.String()
}

func TestExchangeRateSuite(t *testing.T) {
	suite.Run(t, new(ExchangeRateTestSuite))
}

// ---------------------------------------------------------------- tests

func (s *ExchangeRateTestSuite) TestCreateExchangeRate() {
	s.Run("success", func() {
		s.withRoles(shared.WorkspaceRoleAdmin)
		req := newExchangeRateReq()

		rate := s.create(req)

		s.Equal(req.FromCurrency, rate.FromCurrency)
		s.Equal(req.ToCurrency, rate.ToCurrency)
		s.True(req.Rate.Equal(rate.Rate))
		s.Equal(s.User.WorkspaceID, rate.WorkspaceID)
	})

	s.Run("same currency rejected", func() {
		s.withRoles(shared.WorkspaceRoleAdmin)
		req := newExchangeRateReq()
		req.ToCurrency = req.FromCurrency

		s.Request("/exchange-rates", fiber.MethodPost, req)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})

	s.Run("invalid rate rejected", func() {
		s.withRoles(shared.WorkspaceRoleAdmin)
		req := newExchangeRateReq()
		req.Rate = decimal.NewFromInt(0)

		s.Request("/exchange-rates", fiber.MethodPost, req)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})

	s.Run("forbidden for member", func() {
		s.withRoles(shared.WorkspaceRoleMember)
		s.Request("/exchange-rates", fiber.MethodPost, newExchangeRateReq())
		s.AssertStatus(fiber.StatusForbidden)
	})
}

func (s *ExchangeRateTestSuite) TestGetExchangeRate() {
	s.withRoles(shared.WorkspaceRoleAdmin)
	created := s.create(newExchangeRateReq())

	s.Run("success", func() {
		s.Request(exchangeRatePath(created.ID), fiber.MethodGet, nil)
		s.OK()
		s.Equal(created.ID, s.GetResponse().Data.ID)
	})

	s.Run("not found", func() {
		s.Request(exchangeRatePath(uuid.New()), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})
}

func (s *ExchangeRateTestSuite) TestListExchangeRates() {
	s.withRoles(shared.WorkspaceRoleAdmin)

	req1 := newExchangeRateReq()
	req1.FromCurrency = "EUR"
	req1.ToCurrency = "IDR"
	rate1 := s.create(req1)

	req2 := newExchangeRateReq()
	req2.FromCurrency = "JPY"
	req2.ToCurrency = "IDR"
	_ = s.create(req2)

	s.Run("list all", func() {
		s.Request("/exchange-rates", fiber.MethodGet, nil)
		s.OK()
		items := s.PagedResponse().Data
		s.GreaterOrEqual(len(items), 2)
	})

	s.Run("filter by from_currency", func() {
		q := url.Values{}
		q.Set("from_currency", "EUR")
		s.Request("/exchange-rates?"+q.Encode(), fiber.MethodGet, nil)
		s.OK()
		items := s.PagedResponse().Data
		s.Len(items, 1)
		s.Equal(rate1.ID, items[0].ID)
	})

	s.Run("allowed for member", func() {
		s.withRoles(shared.WorkspaceRoleMember)
		s.Request("/exchange-rates", fiber.MethodGet, nil)
		s.OK()
	})
}

func (s *ExchangeRateTestSuite) TestUpdateExchangeRate() {
	s.withRoles(shared.WorkspaceRoleAdmin)
	created := s.create(newExchangeRateReq())

	s.Run("success update rate", func() {
		newRate := decimal.NewFromFloat(15500.50)
		updateReq := model.UpdateExchangeRateRequest{
			Rate: &newRate,
		}

		s.Request(exchangeRatePath(created.ID), fiber.MethodPut, updateReq)
		s.OK()
		s.True(newRate.Equal(s.GetResponse().Data.Rate))
	})

	s.Run("forbidden for member", func() {
		s.withRoles(shared.WorkspaceRoleMember)
		newRate := decimal.NewFromFloat(16000)
		s.Request(exchangeRatePath(created.ID), fiber.MethodPut, model.UpdateExchangeRateRequest{Rate: &newRate})
		s.AssertStatus(fiber.StatusForbidden)
	})
}

func (s *ExchangeRateTestSuite) TestDeleteExchangeRate() {
	s.withRoles(shared.WorkspaceRoleAdmin)
	created := s.create(newExchangeRateReq())

	s.Run("forbidden for member", func() {
		s.withRoles(shared.WorkspaceRoleMember)
		s.Request(exchangeRatePath(created.ID), fiber.MethodDelete, nil)
		s.AssertStatus(fiber.StatusForbidden)
	})

	s.Run("success delete as admin", func() {
		s.withRoles(shared.WorkspaceRoleAdmin)
		s.Request(exchangeRatePath(created.ID), fiber.MethodDelete, nil)
		s.AssertStatus(fiber.StatusNoContent)

		s.Request(exchangeRatePath(created.ID), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})
}

func (s *ExchangeRateTestSuite) TestWorkspaceIsolation() {
	s.withRoles(shared.WorkspaceRoleAdmin)
	created := s.create(newExchangeRateReq())

	// Switch workspace
	otherWS := shared.GenerateID()
	s.User.WorkspaceID = otherWS

	s.Run("cannot get exchange rate from other workspace", func() {
		s.Request(exchangeRatePath(created.ID), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("cannot list exchange rate from other workspace", func() {
		s.Request("/exchange-rates", fiber.MethodGet, nil)
		s.OK()
		for _, item := range s.PagedResponse().Data {
			s.NotEqual(created.ID, item.ID)
		}
	})
}
