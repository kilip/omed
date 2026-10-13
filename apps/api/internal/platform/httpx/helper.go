package httpx

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

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

func DoDelete[ID IDType](c fiber.Ctx, handler func(ctx context.Context, id ID) error) error {
	var id ID
	var err error

	id, err = GetID[ID](c)
	if err != nil {
		return err
	}
	return handler(c, id)
}

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
