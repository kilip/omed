package httpx

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

// DoGetByParamID coordinates extracting a named ID parameter, invoking the entity lookup handler,
// mapping the resulting entity to a response DTO, and rendering an OK response.
func DoGetByParamID[E any, Res any, ID IDType](c fiber.Ctx, name string, h func(c context.Context, id ID) (E, error), toRes func(E) Res) error {
	id, err := GetParamID[ID](c, name)
	if err != nil {
		return err
	}
	e, err := h(c, id)
	if err != nil {
		return err
	}
	return OK(c, toRes(e))
}

// DoGetByID coordinates extracting the ID parameter, invoking the entity lookup handler,
// mapping the resulting entity to a response DTO, and rendering an OK response.
func DoGetByID[E any, Res any, ID IDType](c fiber.Ctx, h func(c context.Context, id ID) (E, error), toRes func(E) Res) error {
	return DoGetByParamID(c, "id", h, toRes)
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

// DoUpdateByParamID coordinates extracting the named parameter, binding the request payload,
// executing the update handler, and rendering the mapped response DTO.
func DoUpdateByParamID[E, Req any, Res any, ID IDType](c fiber.Ctx, name string, handler func(c context.Context, id ID, req Req) (E, error), toRes func(E) Res) error {
	var err error
	var id ID
	var req Req
	var e E

	if id, err = GetParamID[ID](c, name); err != nil {
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

// DoUpdate coordinates extracting the ID, binding the request payload, executing the update handler,
// and rendering the mapped response DTO.
func DoUpdate[E, Req any, Res any, ID IDType](c fiber.Ctx, handler func(c context.Context, id ID, req Req) (E, error), toRes func(E) Res) error {
	return DoUpdateByParamID(c, "id", handler, toRes)
}

// DoDeleteByParamID coordinates extracting the named parameter and executing the deletion handler.
func DoDeleteByParamID[ID IDType](c fiber.Ctx, name string, handler func(ctx context.Context, id ID) error) error {
	var id ID
	var err error

	id, err = GetParamID[ID](c, name)
	if err != nil {
		return err
	}
	return handler(c, id)
}

// DoDelete coordinates extracting the ID and executing the deletion handler.
func DoDelete[ID IDType](c fiber.Ctx, handler func(ctx context.Context, id ID) error) error {
	return DoDeleteByParamID(c, "id", handler)
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
