package server

import (
	"net/http"

	"github.com/DariSorokina/yp-gophermart.git/internal/app"
	"github.com/DariSorokina/yp-gophermart.git/internal/config"
	"github.com/DariSorokina/yp-gophermart.git/internal/cookie"
	"github.com/DariSorokina/yp-gophermart.git/internal/logger"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	handlers   *handlers
	app        *app.App
	flagConfig *config.FlagConfig
	log        *logger.Logger
}

func NewServer(app *app.App, flagConfig *config.FlagConfig, l *logger.Logger) *Server {
	handlers := newHandlers(app, flagConfig, l)
	return &Server{handlers: handlers, app: app, flagConfig: flagConfig, log: l}
}

func (server *Server) newRouter() chi.Router {
	router := chi.NewRouter()
	router.Use(server.log.WithLogging())
	// router.Use(middlewares.CompressorMiddleware())
	router.Use(cookie.SetCookieMiddleware())
	router.Post("/api/user/register", server.handlers.registerHandler)
	router.Post("/api/user/login", server.handlers.loginHandler)
	router.Route("/", func(r chi.Router) {
		r.Use(cookie.CheckCookieMiddleware())
		r.Post("/api/user/orders", server.handlers.postOrderNumberHandler)
		r.Post("/api/user/balance/withdraw", server.handlers.withdrawLoyaltyBonusesHandler)
		r.Get("/api/user/orders", server.handlers.getOrdersNumbersHandler)
		r.Get("/api/user/balance", server.handlers.getLoyaltyBalanceHandler)
		r.Get("/api/user/withdrawals", server.handlers.withdrawalsInfoHandler)
	})
	return router
}

func Run(server *Server) error {
	server.log.Sugar().Infof("Running server on %s", server.flagConfig.FlagRunAddr)
	return http.ListenAndServe(server.flagConfig.FlagRunAddr, server.newRouter())
}
