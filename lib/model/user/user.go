// Contains database types for the user collection.
package user

import "github.com/megakuul/battleshiper/lib/model/rbac"

const USER_COLLECTION = "users"

type ExecutionLimitCounter struct {
	ExpirationTime     int `bson:"expiration_time"`
	PipelineExecutions int `bson:"pipeline_executions"`
}

type User struct {
	Id             string                 `bson:"id"`
	Privileged     bool                   `bson:"privileged"`
	Provider       string                 `bson:"provider"`
	Roles          map[rbac.ROLE]struct{} `bson:"roles"`
	RefreshToken   string                 `bson:"refresh_token"`
	LimitCounter   ExecutionLimitCounter  `bson:"limit_counter"`
	SubscriptionId string                 `bson:"subscription_id"`
}
