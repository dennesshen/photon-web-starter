package server

import "github.com/gofiber/fiber/v2"

type GlobalMiddleware interface {
	GetMiddleware() fiber.Handler
	GetPathPrefix() string
	GetPriority() int
}

var globalMiddleware []GlobalMiddleware

func RegisterGlobalMiddleware(middleware GlobalMiddleware) {
	globalMiddleware = append(globalMiddleware, middleware)
}

func getGlobalMiddleware() []GlobalMiddleware {
	return globalMiddleware
}
