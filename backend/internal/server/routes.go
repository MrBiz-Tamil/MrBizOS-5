package server

import (
	"github.com/gofiber/fiber/v2"
)

// Router defines an interface for modules to register their routes
type Router interface {
	Register(api fiber.Router)
}

// RegisterModules handles the registration of multiple modules
func RegisterModules(api fiber.Router, modules ...Router) {
	for _, module := range modules {
		module.Register(api)
	}
}
