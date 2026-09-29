package authz

type Resource string
type Action string

const (
	ResourceAccounts      Resource = "accounts"
	ResourceWorkspaces    Resource = "workspaces"
	ResourceUsers         Resource = "users"
	ResourceLedgerPeriods Resource = "ledger_periods"
)

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)
