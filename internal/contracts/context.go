package contracts

import "github.com/kilip/omed/internal/entity"

type UserContext struct {
	entity.User
}

type RequestContext interface {
	GetUser() UserContext
}
