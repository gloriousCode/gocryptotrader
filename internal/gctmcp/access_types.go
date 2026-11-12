package gctmcp

// accessPermission names the independently granted private data and control scopes.
type accessPermission uint8

const (
	accessDiagnostics accessPermission = 1 << iota
	accessAccountData
	accessOrderManagement
	accessWithdrawals
	accessSystemControl
)

// methodPolicy keeps the authority for one RPC in a single classification.
type methodPolicy struct {
	permissions accessPermission
	execution   bool
}
