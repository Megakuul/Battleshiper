// contains list with all roles used over the application.
package role

type ROLE int

const (
	USER ROLE = iota
	SUPPORT
	SUBSCRIPTION_MANAGER
	ROLE_MANAGER
	ADMIN
)
