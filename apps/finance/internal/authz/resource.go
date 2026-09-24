package authz

type Resource string
type Action string

const (
	ResourceAccounts  Resource = "accounts"
	ResourceWorkspace Resource = "workspace"
)

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)
