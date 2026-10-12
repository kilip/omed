package httpx

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

func DoGetByID[Res any, ID IDType](c fiber.Ctx, handler func(c context.Context, id ID) (Res, error)) error {
	var id ID
	var res Res
	var err error

	if id, err = GetID[ID](c); err != nil {
		return err
	}

	res, err = handler(c, id)
	if err != nil {
		return err
	}

	return OK(c, res)
}

func DoCreate[Req any, Res any](c fiber.Ctx, handler func(c context.Context, req Req) (Res, error)) error {
	var err error
	var req Req
	var res Res

	if err = c.Bind().Body(&req); err != nil {
		return err
	}

	if res, err = handler(c, req); err != nil {
		return err
	}

	return Created(c, res)
}

func DoUpdate[Req any, Res any, ID IDType](c fiber.Ctx, handler func(c context.Context, id ID, req Req) (Res, error)) error {
	var err error
	var id ID
	var req Req
	var res Res

	if id, err = GetID[ID](c); err != nil {
		return err
	}

	if err = c.Bind().Body(&req); err != nil {
		return err
	}

	if res, err = handler(c, id, req); err != nil {
		return err
	}

	return OK(c, res)
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

func DoSearch[Req any, Res any](c fiber.Ctx, handler func(ctx context.Context, req Req) (Res, string, error)) error {
	var err error
	var req Req
	var res Res
	var cursor string

	if err = c.Bind().Body(&req); err != nil {
		return err
	}

	if res, cursor, err = handler(c, req); err != nil {
		return err
	}

	return WithCursor(c, res, cursor)
}
