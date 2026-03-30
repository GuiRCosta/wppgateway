package response

import "github.com/gofiber/fiber/v2"

type Meta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    *T     `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK[T any](c *fiber.Ctx, data T) error {
	return c.JSON(APIResponse[T]{
		Success: true,
		Data:    &data,
	})
}

func Created[T any](c *fiber.Ctx, data T) error {
	return c.Status(fiber.StatusCreated).JSON(APIResponse[T]{
		Success: true,
		Data:    &data,
	})
}

func List[T any](c *fiber.Ctx, data T, meta Meta) error {
	return c.JSON(APIResponse[T]{
		Success: true,
		Data:    &data,
		Meta:    &meta,
	})
}

func Err(c *fiber.Ctx, status int, code string, message string) error {
	return c.Status(status).JSON(APIResponse[any]{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	})
}

func ErrBadRequest(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusBadRequest, "bad_request", message)
}

func ErrUnauthorized(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusUnauthorized, "unauthorized", message)
}

func ErrForbidden(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusForbidden, "forbidden", message)
}

func ErrNotFound(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusNotFound, "not_found", message)
}

func ErrConflict(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusConflict, "conflict", message)
}

func ErrInternal(c *fiber.Ctx, message string) error {
	return Err(c, fiber.StatusInternalServerError, "internal_error", message)
}

func ErrRateLimited(c *fiber.Ctx) error {
	return Err(c, fiber.StatusTooManyRequests, "rate_limited", "Rate limit exceeded")
}
