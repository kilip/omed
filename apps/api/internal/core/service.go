package core

import (
	"context"
)

type ServiceRepository[IDType any, Res any, CreateRequest any, UpdateRequest any, SearchRequest any] interface {
	GetByID(c context.Context, id IDType) (Res, error)
	Search(c context.Context, req SearchRequest) ([]Res, string, error)
	Create(c context.Context, req CreateRequest) (Res, error)
	Update(c context.Context, id IDType, req UpdateRequest) (Res, error)
	Delete(c context.Context, id IDType) error
}

type Service[IDType any, Res, CreateRequest any, UpdateRequest any, SearchRequest any] struct {
	repo ServiceRepository[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]
}

func (s *Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) SetRepository(repo ServiceRepository[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) {
	s.repo = repo
}

func (s Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) GetByID(c context.Context, id IDType) (Res, error) {
	return s.repo.GetByID(c, id)
}

func (s Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) Search(c context.Context, req SearchRequest) ([]Res, string, error) {
	return s.repo.Search(c, req)
}

func (s Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) Create(c context.Context, req CreateRequest) (Res, error) {
	return s.repo.Create(c, req)
}

func (s Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) Update(c context.Context, id IDType, req UpdateRequest) (Res, error) {
	var res Res
	if _, err := s.repo.GetByID(c, id); err != nil {
		return res, err
	}
	return s.repo.Update(c, id, req)
}

func (s Service[IDType, Res, CreateRequest, UpdateRequest, SearchRequest]) Delete(c context.Context, id IDType) error {
	if _, err := s.repo.GetByID(c, id); err != nil {
		return err
	}
	return s.repo.Delete(c, id)
}
