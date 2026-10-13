package core

import (
	"context"
)

// ServiceRepository defines generic data-access operations required by the generic Service.
type ServiceRepository[ID, Entity, Create, Update, Search any] interface {
	GetByID(c context.Context, id ID) (Entity, error)
	Search(c context.Context, req Search) ([]Entity, string, error)
	Create(c context.Context, req Create) (Entity, error)
	Update(c context.Context, id ID, req Update) (Entity, error)
	Delete(c context.Context, id ID) error
}

// Service provides a reusable generic CRUD implementation backed by a ServiceRepository.
type Service[ID, Entity, Create, Update, Search any] struct {
	repo ServiceRepository[ID, Entity, Create, Update, Search]
}

// SetRepository configures or replaces the underlying repository backing the service.
func (s *Service[ID, E, C, U, S]) SetRepository(repo ServiceRepository[ID, E, C, U, S]) {
	s.repo = repo
}

// GetByID retrieves a single entity by its unique identifier.
func (s Service[ID, E, C, U, S]) GetByID(c context.Context, id ID) (E, error) {
	return s.repo.GetByID(c, id)
}

// Search queries entities based on search criteria and returns matching items along with a next page cursor.
func (s Service[ID, E, C, U, S]) Search(c context.Context, req S) ([]E, string, error) {
	return s.repo.Search(c, req)
}

// Create persists a new entity using the provided creation payload.
func (s Service[ID, E, C, U, S]) Create(c context.Context, req C) (E, error) {
	return s.repo.Create(c, req)
}

// Update modifies an existing entity identified by id with the provided update payload.
// It returns an error if the entity does not exist or if updating fails.
func (s Service[ID, E, C, U, S]) Update(c context.Context, id ID, req U) (E, error) {
	var entity E
	if _, err := s.repo.GetByID(c, id); err != nil {
		return entity, err
	}
	return s.repo.Update(c, id, req)
}

// Delete removes an existing entity identified by id.
// It returns an error if the entity does not exist or if deletion fails.
func (s Service[ID, E, C, U, S]) Delete(c context.Context, id ID) error {
	if _, err := s.repo.GetByID(c, id); err != nil {
		return err
	}
	return s.repo.Delete(c, id)
}

// NewService constructs a new generic Service instance backed by the given repository.
func NewService[ID, E, C, U, S any](repo ServiceRepository[ID, E, C, U, S]) Service[ID, E, C, U, S] {
	return Service[ID, E, C, U, S]{repo: repo}
}
