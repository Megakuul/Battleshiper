// contains list with all roles used over the application.
package rbac

type ROLE int

const (
	USER ROLE = iota
	SUPPORT
	MAINTAINER
	SUBSCRIPTION_MANAGER
	ROLE_MANAGER
)

type ACCESS int

const (
	READ_USER ACCESS = iota
	READ_PROJECT
	WRITE_USER
	WRITE_PROJECT
	WRITE_SUBSCRIPTION
	WRITE_ROLE
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
		WRITE_USER:    struct{}{},
		WRITE_PROJECT: struct{}{},
	},
	SUBSCRIPTION_MANAGER: {
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
		if role > USER {
			return true
		}
	}
	return false
}
