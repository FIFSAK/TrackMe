package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"TrackMe/docs"
	"TrackMe/internal/config"
	"TrackMe/internal/handler/http"
	"TrackMe/internal/service/track"
	"TrackMe/pkg/jwt"
	"TrackMe/pkg/server/router"
)

type Dependencies struct {
	Configs      config.Configs
	TrackService *track.Service
}

// Configuration is an alias for a function that will take in a pointer to a Handler and modify it
type Configuration func(h *Handler) error

// Handler is an implementation of the Handler
type Handler struct {
	dependencies Dependencies

	HTTP *chi.Mux
}

// New takes a variable amount of Configuration functions and returns a new Handler
// Each Configuration will be called in the order they are passed in
func New(d Dependencies, configs ...Configuration) (h *Handler, err error) {
	// Create the handler
	h = &Handler{
		dependencies: d,
	}

	// Apply all Configurations passed in
	for _, cfg := range configs {
		// Pass the service into the configuration function
		if err = cfg(h); err != nil {
			return
		}
	}

	return
}

// WithHTTPHandler applies a http handler to the Handler
func WithHTTPHandler() Configuration {
	return func(h *Handler) (err error) {
		// Create the http handler, if we needed parameters, such as connection strings they could be inputted here
		h.HTTP = router.New()

		h.HTTP.Use(middleware.Timeout(h.dependencies.Configs.APP.Timeout))

		basePath := h.dependencies.Configs.APP.Path

		// Init swagger handler
		docs.SwaggerInfo.BasePath = h.dependencies.Configs.APP.Path
		h.HTTP.Get("/swagger/*", httpSwagger.WrapHandler)

		// Create JWT token manager
		tokenManager := jwt.NewTokenManager(h.dependencies.Configs.JWT.SecretKey)

		// Init service handlers
		authHandler := http.NewAuthHandler(h.dependencies.TrackService, tokenManager)
		clientHandler := http.NewClientHandler(h.dependencies.TrackService, tokenManager)
		userHandler := http.NewUserHandler(h.dependencies.TrackService, tokenManager)
		metricHandler := http.NewMetricHandler(h.dependencies.TrackService, tokenManager)

		h.HTTP.Route(basePath+"/", func(r chi.Router) {
			r.Mount("/auth", authHandler.Routes())
			r.Mount("/clients", clientHandler.Routes())
			r.Mount("/users", userHandler.Routes())
			r.Mount("/metrics", metricHandler.Routes())
		})
		return
	}
}
