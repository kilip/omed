package core

import (
	"context"
)

type ServiceRepository[ID, Entity, Create, Update, Search any] interface {
	GetByID(c context.Context, id ID) (Entity, error)
	Search(c context.Context, req Search) ([]Entity, string, error)
	Create(c context.Context, req Create) (Entity, error)
	Update(c context.Context, id ID, req Update) (Entity, error)
	Delete(c context.Context, id ID) error
}

type Service[ID, Entity, Create, Update, Search any] struct {
	repo ServiceRepository[ID, Entity, Create, Update, Search]
}

func (s *Service[ID, E, C, U, S]) SetRepository(repo ServiceRepository[ID, E, C, U, S]) {
	s.repo = repo
}

func (s Service[ID, E, C, U, S]) GetByID(c context.Context, id ID) (E, error) {
	return s.repo.GetByID(c, id)
}

func (s Service[ID, E, C, U, S]) Search(c context.Context, req S) ([]E, string, error) {
	return s.repo.Search(c, req)
}

func (s Service[ID, E, C, U, S]) Create(c context.Context, req C) (E, error) {
	return s.repo.Create(c, req)
}

func (s Service[ID, E, C, U, S]) Update(c context.Context, id ID, req U) (E, error) {
	var entity E
	if _, err := s.repo.GetByID(c, id); err != nil {
		return entity, err
	}
	return s.repo.Update(c, id, req)
}

func (s Service[ID, E, C, U, S]) Delete(c context.Context, id ID) error {
	if _, err := s.repo.GetByID(c, id); err != nil {
		return err
	}
	return s.repo.Delete(c, id)
}

func NewService[ID, E, C, U, S any](repo ServiceRepository[ID, E, C, U, S]) Service[ID, E, C, U, S] {
	return Service[ID, E, C, U, S]{repo: repo}
}
