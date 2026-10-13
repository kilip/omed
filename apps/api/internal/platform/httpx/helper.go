package httpx

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

// DoGetByID coordinates extracting the ID parameter, invoking the entity lookup handler,
// mapping the resulting entity to a response DTO, and rendering an OK response.
func DoGetByID[E any, Res any, ID IDType](c fiber.Ctx, h func(c context.Context, id ID) (E, error), toRes func(E) Res) error {
	id, err := GetID[ID](c)
	if err != nil {
		return err
	}
	e, err := h(c, id)
	if err != nil {
		return err
	}
	return OK(c, toRes(e))
}

// DoCreate coordinates parsing and binding the request body, invoking the creation handler,
// mapping the created entity to a response DTO, and rendering a Created response.
func DoCreate[E, Req any, Res any](c fiber.Ctx, handler func(c context.Context, req Req) (E, error), toRes func(E) Res) error {
	var err error
	var req Req
	var e E

	if err = c.Bind().Body(&req); err != nil {
		return err
	}

	if e, err = handler(c, req); err != nil {
		return err
	}

	return Created(c, toRes(e))
}

// DoUpdate coordinates extracting the ID, binding the request payload, executing the update handler,
// and rendering the mapped response DTO.
func DoUpdate[E, Req any, Res any, ID IDType](c fiber.Ctx, handler func(c context.Context, id ID, req Req) (E, error), toRes func(E) Res) error {
	var err error
	var id ID
	var req Req
	var e E

	if id, err = GetID[ID](c); err != nil {
		return err
	}

	if err = c.Bind().Body(&req); err != nil {
		return err
	}

	if e, err = handler(c, id, req); err != nil {
		return err
	}

	return OK(c, toRes(e))
}

// DoDelete coordinates extracting the ID and executing the deletion handler.
func DoDelete[ID IDType](c fiber.Ctx, handler func(ctx context.Context, id ID) error) error {
	var id ID
	var err error

	id, err = GetID[ID](c)
	if err != nil {
		return err
	}
	return handler(c, id)
}

// DoSearch coordinates binding the search criteria payload, invoking the search handler,
// mapping each entity to a response DTO, and rendering a cursor-paginated response.
func DoSearch[E, Req, Res any](c fiber.Ctx, h func(context.Context, Req) ([]E, string, error), toRes func(E) Res) error {
	var req Req
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := h(c, req)
	if err != nil {
		return err
	}
	out := make([]Res, len(items))
	for i, it := range items {
		out[i] = toRes(it)
	}
	return WithCursor(c, out, cursor)
}
