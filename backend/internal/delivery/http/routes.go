package http

import (
	"MRG/internal/config"
	v1 "MRG/internal/delivery/http/handlers/v1"
	"MRG/internal/delivery/http/middleware"
	"MRG/internal/domain/user"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	authHandler *v1.AuthHandler,
	userHandler *v1.UserHandler,
	leadHandler *v1.LeadHandler,
	signalHandler *v1.SignalHandler,
	settingsHandler *v1.SettingsHandler,
	knowledgeHandler *v1.KnowledgeHandler,
	accountHandler *v1.AccountHandler,
	userRepo user.Repository,
	cfg *config.Config,
) {
	api := app.Group("/api")
	g := api.Group("/v1")

	g.Post("/auth/login", authHandler.Login)
	g.Post("/auth/refresh", authHandler.Refresh)
	g.Get("/auth/invites/:token", authHandler.GetInvite)
	g.Post("/auth/invites/:token/accept", authHandler.AcceptInvite)

	if accountHandler != nil {
		accounts := g.Group("/accounts", middleware.AuthRequired(cfg), middleware.PermissionRequired(user.PermissionManageChats, userRepo))
		accounts.Get("/", accountHandler.List)
		accounts.Post("/", accountHandler.Add)
		accounts.Post("/:id/auth", accountHandler.Authenticate)
		accounts.Post("/:id/code", accountHandler.ProvideCode)
		accounts.Post("/:id/password", accountHandler.ProvidePassword)
		accounts.Post("/:id/start", accountHandler.Start)
		accounts.Post("/:id/stop", accountHandler.Stop)
		accounts.Post("/:id/restart", accountHandler.Restart)
		accounts.Delete("/:id", accountHandler.Delete)
	}

	users := g.Group("/users", middleware.AuthRequired(cfg), middleware.PermissionRequired(user.PermissionManageUsers, userRepo))
	users.Get("/", userHandler.ListUsers)
	users.Post("/", userHandler.CreateUser)
	users.Post("/invites", userHandler.CreateInvite)
	users.Get("/:id", userHandler.GetUser)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", userHandler.DeleteUser)
	users.Post("/:id/roles", userHandler.AssignRole)
	users.Delete("/:id/roles", userHandler.RevokeRole)

	leads := g.Group("/leads", middleware.AuthRequired(cfg), middleware.PermissionRequired(user.PermissionViewLeads, userRepo))
	leads.Get("/", leadHandler.GetLeads)
	leads.Get("/stats", leadHandler.GetStats)
	leads.Get("/:id/brief", leadHandler.GetLeadBrief)
	leads.Patch("/:id/status", leadHandler.UpdateStatus)
	leads.Delete("/:id", leadHandler.Delete)
	leads.Post("/:id/approve", leadHandler.Approve)
	leads.Post("/:id/reject", leadHandler.Reject)
	leads.Put("/:id/merchant", leadHandler.SetMerchant)

	if signalHandler != nil {
		signals := g.Group("/signals", middleware.AuthRequired(cfg), middleware.PermissionRequired(user.PermissionViewLeads, userRepo))
		signals.Get("/stats", signalHandler.GetStats)
		signals.Get("/inbox", signalHandler.GetInbox)
		signals.Get("/sender/:senderID", signalHandler.GetSenderHistory)
		signals.Post("/:id/feedback", signalHandler.FeedbackSignal)
		signals.Post("/:id/flag", signalHandler.FlagSignal)
		signals.Post("/:id/bind-merchant", signalHandler.BindContact)
	}

	settingsGroup := g.Group("/settings", middleware.AuthRequired(cfg), middleware.PermissionRequired(user.PermissionManageUsers, userRepo))
	settingsGroup.Get("/", settingsHandler.GetSettings)
	settingsGroup.Put("/", settingsHandler.UpdateSettings)
	if knowledgeHandler != nil {
		settingsGroup.Post("/knowledge/import", knowledgeHandler.ImportCSV)
	}
}
