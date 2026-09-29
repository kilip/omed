package authz

type Resource string
type Action string

const (
	ResourceAccounts   Resource = "accounts"
	ResourceWorkspaces Resource = "workspaces"
	ResourceUsers      Resource = "users"
)

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)
