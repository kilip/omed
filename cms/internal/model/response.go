package model

type Resource[T any] struct {
	Data T `json:"data"`
	Meta struct {
		Uri string `json:"uri"`
	} `json:"meta,omitempty"`
}

type Resources[T any] struct {
	Data			T	`json:"data"`
	Meta struct {
		Page 	*PageMetadata `json:"page,omitempty"`
	} `json:"meta"`
	
}

type PagedResources[T any] struct {
	Data         []T          `json:"data,omitempty"`
}

type PageMetadata struct {
	Page      	int   `json:"current"`
	MaxPerPage	int   `json:"maxPerPage"`
	TotalItem 	int64 `json:"rows"`
	TotalPage 	int64 `json:"pages"`
	Next		string `json:"next"`
	Previous 	string `json:"previous"`
	First		string `json:"first"`
	Last		string `json:"last"`
}
