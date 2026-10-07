package authz

type Resource string
type Action string

const (
	ResourceAccounts      Resource = "accounts"
	ResourceWorkspaces    Resource = "workspaces"
	ResourceUsers         Resource = "users"
	ResourceLedgerPeriods Resource = "ledger_periods"
	ResourceEntries       Resource = "entries"
	ResourceExchangeRates Resource = "exchange_rates"
	ResourceAttachments   Resource = "attachments"
	ResourceReports       Resource = "reports"
	ResourceContacts      Resource = "contacts"
	ResourceTaxRates      Resource = "tax_rates"
	ResourceInvoices      Resource = "invoices"
	ResourcePayments      Resource = "payments"
)

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)
