// contains list with all roles used over the application.
package rbac

type ROLE string

const (
	USER                 ROLE = "USER"
	SUPPORT              ROLE = "SUPPORT"
	MAINTAINER           ROLE = "MAINTAINER"
	SUBSCRIPTION_MANAGER ROLE = "SUBSCRIPTION_MANAGER"
	ROLE_MANAGER         ROLE = "ROLE_MANAGER"
)

type ACCESS string

const (
	READ_USER          ACCESS = "READ_USER"
	READ_PROJECT       ACCESS = "READ_PROJECT"
	READ_LOGS          ACCESS = "READ_LOGS"
	READ_SUBSCRIPTION  ACCESS = "READ_SUBSCRIPTION"
	WRITE_USER         ACCESS = "WRITE_USER"
	WRITE_PROJECT      ACCESS = "WRITE_PROJECT"
	WRITE_SUBSCRIPTION ACCESS = "WRITE_SUBSCRIPTION"
	WRITE_ROLE         ACCESS = "WRITE_ROLE"
)

var RBAC_MAP = map[ROLE]map[ACCESS]struct{}{
	USER: {},
	SUPPORT: {
		READ_USER:    struct{}{},
		READ_PROJECT: struct{}{},
	},
	MAINTAINER: {
		READ_USER:     struct{}{},
		READ_PROJECT:  struct{}{},
		READ_LOGS:     struct{}{},
		WRITE_USER:    struct{}{},
		WRITE_PROJECT: struct{}{},
	},
	SUBSCRIPTION_MANAGER: {
		READ_SUBSCRIPTION:  struct{}{},
		WRITE_SUBSCRIPTION: struct{}{},
	},
	ROLE_MANAGER: {
		WRITE_ROLE: struct{}{},
	},
}

// CheckPermission checks if the provided roles contain the specified access.
func CheckPermission(roles map[ROLE]struct{}, access ACCESS) bool {
	for k := range roles {
		if _, exists := RBAC_MAP[k][access]; exists {
			return true
		}
	}
	return false
}

// IsPrivileged checks if the user is privileged (has elevated permissions).
func IsPrivileged(roles map[ROLE]struct{}) bool {
	for role := range roles {
		if role != "" && role != USER {
			return true
		}
	}
	return false
}
