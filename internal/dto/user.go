package dto

import "github.com/google/uuid"


type UserData struct {
	ID     uuid.UUID `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Email  string `json:"email,omitempty"`
}

type UserListRequest struct {
	Sorts SortType
	Filters FilterType
}

type UserRequest struct {
	ID       uuid.UUID `json:"-"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Name     string `json:"name,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}
