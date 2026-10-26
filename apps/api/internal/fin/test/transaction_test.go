package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/internal/base"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type TransactionApiSuite struct {
	testutil.ApiTestSuite[model.TransactionResponse]
}

func (s *TransactionApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *TransactionApiSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string) int64 {
	client := testutil.GetEntCli()
	acc, err := client.Account.Create().
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(name).
		SetType(account.Type(accType)).
		SetNormalBalance(account.NormalBalance(normalBalance)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return acc.ID
}

func (s *TransactionApiSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int) int64 {
	client := testutil.GetEntCli()
	p, err := client.Period.Create().
		SetWorkspaceID(wsID).
		SetName(name).
		SetStartDate(start).
		SetEndDate(end).
		SetFiscalYear(year).
		SetStatus(period.Status(domain.PeriodStatusOpen)).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return p.ID
}

func (s *TransactionApiSuite) getListResponse() httpx.WebResponse[[]model.TransactionResponse] {
	var res httpx.WebResponse[[]model.TransactionResponse]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *TransactionApiSuite) Test_Create_InlineEntry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "3000", "Owner Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit)
	periodID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Initial capital injection",
		Reference:       "REF-001",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: "Capital deposit",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: 5000000},
				{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: 5000000},
			},
		},
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("Initial capital injection", res.Data.Description)
	s.Equal("REF-001", res.Data.Reference)
	s.Equal("IDR", res.Data.Currency)
	s.Equal("MANUAL", res.Data.Source)
	s.Require().Len(res.Data.Entries, 1)
	s.Require().Len(res.Data.Entries[0].Postings, 2)
}

func (s *TransactionApiSuite) Test_Create_HeaderOnly_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Header only transaction",
		Reference:       "REF-HEADER",
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("Header only transaction", res.Data.Description)
	s.Empty(res.Data.Entries)
}

func (s *TransactionApiSuite) Test_GetByID_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Transaction to get",
		Reference:       "REF-GET",
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(created.ID, res.Data.ID)
	s.Equal("REF-GET", res.Data.Reference)
}

func (s *TransactionApiSuite) Test_Search_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "First tx",
		Reference:       "REF-SEARCH-1",
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Second tx",
		Reference:       "REF-SEARCH-2",
	})
	s.Created()

	// Search all in workspace
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/search", ws.ID), fiber.MethodPost, model.SearchTransactionRequest{})
	s.OK()

	listRes := s.getListResponse()
	s.Len(listRes.Data, 2)

	// Search by exact reference
	refQuery := "REF-SEARCH-1"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/search", ws.ID), fiber.MethodPost, model.SearchTransactionRequest{
		Reference: &refQuery,
	})
	s.OK()

	filteredRes := s.getListResponse()
	s.Len(filteredRes.Data, 1)
	s.Equal("REF-SEARCH-1", filteredRes.Data[0].Reference)
}

func (s *TransactionApiSuite) Test_Update_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Before update",
		Reference:       "OLD-REF",
	})
	s.Created()
	created := s.GetResponse().Data

	newDesc := "After update"
	newRef := "NEW-REF"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateTransactionRequest{
		Description: &newDesc,
		Reference:   &newRef,
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("After update", res.Data.Description)
	s.Equal("NEW-REF", res.Data.Reference)
}

func (s *TransactionApiSuite) Test_Delete_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "To delete",
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// GET after delete returns 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *TransactionApiSuite) Test_Update_BlockedIfPosted() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "3000", "Owner Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit)
	periodID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Capital injection",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: "Capital deposit",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: 100000},
				{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: 100000},
			},
		},
	})
	s.Created()
	tx := s.GetResponse().Data
	entryID := tx.Entries[0].ID

	// Post the entry
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	// Try to update transaction header -> 409 Conflict
	newDesc := "Should fail"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodPut, model.UpdateTransactionRequest{
		Description: &newDesc,
	})
	s.AssertStatus(fiber.StatusConflict)

	// Try to delete transaction -> 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *TransactionApiSuite) uploadAttachment(wid, txID int64, filename string, content []byte) {
	s.T().Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		s.Require().NoError(err)
		_, err = part.Write(content)
		s.Require().NoError(err)
	}
	s.Require().NoError(w.Close())

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/fin/transactions/%d/attachments", wid, txID), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	}

	resp, err := testutil.GetState().FiberApp.Test(req)
	s.Require().NoError(err)
	s.HttpResponse = resp
}

func (s *TransactionApiSuite) Test_Attachments_UploadGetDelete_And_PostedLock() {
	ws := s.ensureWorkspace("Finance Attachments WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "3000", "Owner Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit)
	periodID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Receipt attached tx",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: "Receipt entry",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: 250000},
				{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: 250000},
			},
		},
	})
	s.Created()
	tx := s.GetResponse().Data

	// 1. Disallowed extension -> 422
	s.uploadAttachment(ws.ID, tx.ID, "script.sh", []byte("#!/bin/sh"))
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// 2. Upload two valid attachments
	s.uploadAttachment(ws.ID, tx.ID, "receipt-1.pdf", []byte("%PDF-1.4 receipt 1"))
	s.Created()
	att1 := testutil.DecodeOther[coremodel.FileResponse](s).Data
	s.Equal("receipt-1.pdf", att1.Name)
	s.Equal(domain.TransactionEntityType, att1.EntityType)
	s.Equal(tx.ID, att1.EntityID)

	s.uploadAttachment(ws.ID, tx.ID, "receipt-2.png", []byte("png-bytes"))
	s.Created()
	att2 := testutil.DecodeOther[coremodel.FileResponse](s).Data

	// 3. GET transaction includes both attachments
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Require().Len(fetched.Attachments, 2)

	// 4. Delete second attachment while still draft
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d/attachments/%d", ws.ID, tx.ID, att2.ID), fiber.MethodDelete, nil)
	s.OK()

	// 5. Post entry -> uploading or deleting attachment returns 409 Conflict
	entryID := tx.Entries[0].ID
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	s.uploadAttachment(ws.ID, tx.ID, "receipt-3.pdf", []byte("%PDF-1.4 receipt 3"))
	s.AssertStatus(fiber.StatusConflict)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d/attachments/%d", ws.ID, tx.ID, att1.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *TransactionApiSuite) Test_DeleteTransaction_CleansUpAttachmentsInS3() {
	ws := s.ensureWorkspace("Finance Cleanup WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Draft tx with attachment to delete",
	})
	s.Created()
	tx := s.GetResponse().Data

	s.uploadAttachment(ws.ID, tx.ID, "invoice.pdf", []byte("%PDF-1.4 cleanup test"))
	s.Created()
	att := testutil.DecodeOther[coremodel.FileResponse](s).Data

	// Delete transaction
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodDelete, nil)
	s.OK()

	// Verify S3 object is removed and core file endpoint returns 404
	st := testutil.GetStorage()
	_, err := st.StatObject(context.Background(), att.Bucket, att.Key)
	s.ErrorIs(err, s3x.ErrObjectNotFound)

	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, att.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func TestTransactionApiSuite(t *testing.T) {
	suite.Run(t, new(TransactionApiSuite))
}
