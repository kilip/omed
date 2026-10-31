package fin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/controller"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod FinModule, r fiber.Router, az *authz.Authorizer) {
	accounts := r.Group("/workspaces/:wid/fin/accounts")
	accountCtl := controller.NewAccountController(mod.AccountService, mod.EntryService)

	accounts.Post("/search", az.Require(domain.ObjAccount, "read"), accountCtl.Search)
	accounts.Post("/seed", az.Require(domain.ObjAccount, "create"), accountCtl.Seed)
	accounts.Get("/templates/:template", az.Require(domain.ObjAccount, "read"), accountCtl.PreviewTemplate)
	accounts.Post("/", az.Require(domain.ObjAccount, "create"), accountCtl.Create)
	accounts.Get("/:id/ledger", az.Require(domain.ObjAccount, "read"), accountCtl.Ledger)
	accounts.Get("/:id", az.Require(domain.ObjAccount, "read"), accountCtl.GetByID)
	accounts.Put("/:id", az.Require(domain.ObjAccount, "update"), accountCtl.Update)
	accounts.Delete("/:id", az.Require(domain.ObjAccount, "delete"), accountCtl.Delete)

	periods := r.Group("/workspaces/:wid/fin/periods")
	periodCtl := controller.NewPeriodController(mod.PeriodService)

	periods.Post("/search", az.Require(domain.ObjPeriod, "read"), periodCtl.Search)
	periods.Post("/", az.Require(domain.ObjPeriod, "create"), periodCtl.Create)
	periods.Get("/:id", az.Require(domain.ObjPeriod, "read"), periodCtl.GetByID)
	periods.Put("/:id", az.Require(domain.ObjPeriod, "update"), periodCtl.Update)
	periods.Delete("/:id", az.Require(domain.ObjPeriod, "delete"), periodCtl.Delete)
	periods.Post("/:id/close", az.Require(domain.ObjPeriod, "update"), periodCtl.Close)
	periods.Post("/:id/reopen", az.Require(domain.ObjPeriod, "update"), periodCtl.Reopen)
	periods.Post("/:id/revalue", az.Require(domain.ObjPeriod, "update"), periodCtl.Revalue)

	transactions := r.Group("/workspaces/:wid/fin/transactions")
	txCtl := controller.NewTransactionController(mod.TransactionService)

	transactions.Post("/search", az.Require(domain.ObjTransaction, "read"), txCtl.Search)
	transactions.Post("/", az.Require(domain.ObjTransaction, "create"), txCtl.Create)
	transactions.Get("/:id", az.Require(domain.ObjTransaction, "read"), txCtl.GetByID)
	transactions.Put("/:id", az.Require(domain.ObjTransaction, "update"), txCtl.Update)
	transactions.Delete("/:id", az.Require(domain.ObjTransaction, "delete"), txCtl.Delete)
	transactions.Post("/:id/attachments", az.Require(domain.ObjTransaction, "update"), txCtl.UploadAttachment)
	transactions.Delete("/:id/attachments/:fileId", az.Require(domain.ObjTransaction, "update"), txCtl.DeleteAttachment)

	cashFlowCtl := controller.NewCashFlowController(mod.CashFlowService)
	expense := r.Group("/workspaces/:wid/fin/expense")
	expense.Post("/", az.Require(domain.ObjTransaction, "create"), cashFlowCtl.CreateExpense)
	expense.Post("/search", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.SearchExpenses)
	expense.Get("/:id", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.GetExpense)
	expense.Put("/:id", az.Require(domain.ObjTransaction, "update"), cashFlowCtl.UpdateExpense)

	income := r.Group("/workspaces/:wid/fin/income")
	income.Post("/", az.Require(domain.ObjTransaction, "create"), cashFlowCtl.CreateIncome)
	income.Post("/search", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.SearchIncomes)
	income.Get("/:id", az.Require(domain.ObjTransaction, "read"), cashFlowCtl.GetIncome)
	income.Put("/:id", az.Require(domain.ObjTransaction, "update"), cashFlowCtl.UpdateIncome)

	entries := r.Group("/workspaces/:wid/fin/entries")
	entryCtl := controller.NewEntryController(mod.EntryService)

	entries.Post("/search", az.Require(domain.ObjEntry, "read"), entryCtl.Search)
	entries.Post("/", az.Require(domain.ObjEntry, "create"), entryCtl.Create)
	entries.Get("/:id", az.Require(domain.ObjEntry, "read"), entryCtl.GetByID)
	entries.Put("/:id", az.Require(domain.ObjEntry, "update"), entryCtl.Update)
	entries.Delete("/:id", az.Require(domain.ObjEntry, "delete"), entryCtl.Delete)
	entries.Post("/:id/post", az.Require(domain.ObjEntry, "update"), entryCtl.Post)
	entries.Post("/:id/reverse", az.Require(domain.ObjEntry, "create"), entryCtl.Reverse)
	entries.Post("/:id/void", az.Require(domain.ObjEntry, "delete"), entryCtl.Void)

	currencyCtl := controller.NewCurrencyController(mod.CurrencyService)
	currencies := r.Group("/workspaces/:wid/fin/currencies")
	currencies.Get("/", az.Require(domain.ObjCurrency, "read"), currencyCtl.List)

	settingCtl := controller.NewSettingController(mod.SettingService)
	settings := r.Group("/workspaces/:wid/fin/settings")
	settings.Get("/", az.Require(domain.ObjAccount, "read"), settingCtl.GetSettings)
	settings.Put("/", az.Require(domain.ObjAccount, "update"), settingCtl.UpdateSettings)
	settings.Put("/base-currency", az.Require(domain.ObjCurrency, "update"), currencyCtl.UpdateBaseCurrency)
	settings.Get("/fx-accounts", az.Require(domain.ObjAccount, "read"), settingCtl.GetFXAccounts)
	settings.Put("/fx-accounts", az.Require(domain.ObjAccount, "update"), settingCtl.UpdateFXAccounts)

	exchangeRates := r.Group("/workspaces/:wid/fin/exchange-rates")
	rateCtl := controller.NewExchangeRateController(mod.ExchangeRateService)
	exchangeRates.Post("/search", az.Require(domain.ObjExchangeRate, "read"), rateCtl.Search)
	exchangeRates.Post("/", az.Require(domain.ObjExchangeRate, "create"), rateCtl.Create)
	exchangeRates.Get("/lookup", az.Require(domain.ObjExchangeRate, "read"), rateCtl.Lookup)
	exchangeRates.Get("/:id", az.Require(domain.ObjExchangeRate, "read"), rateCtl.GetByID)
	exchangeRates.Put("/:id", az.Require(domain.ObjExchangeRate, "update"), rateCtl.Update)
	exchangeRates.Delete("/:id", az.Require(domain.ObjExchangeRate, "delete"), rateCtl.Delete)

	reportCtl := controller.NewReportController(mod.ReportService)
	reports := r.Group("/workspaces/:wid/fin/reports")
	reports.Post("/trial-balance", az.Require(domain.ObjAccount, "read"), reportCtl.TrialBalance)
	reports.Get("/summary", az.Require(domain.ObjAccount, "read"), reportCtl.Summary)
}
