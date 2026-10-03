package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/branding"
	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/store"
	"github.com/m1k1o/neko-rooms/internal/types"
)

type ApiManagerCtx struct {
	logger    zerolog.Logger
	rooms     types.RoomManager
	pull      types.PullManager
	auth      *auth.Manager
	branding  *branding.Manager
	community *community.Service
	store     *store.Store
}

func New(rooms types.RoomManager, pull types.PullManager, authManager *auth.Manager, brandingManager *branding.Manager, communityService *community.Service) *ApiManagerCtx {
	return &ApiManagerCtx{
		logger:    log.With().Str("module", "api").Logger(),
		rooms:     rooms,
		pull:      pull,
		auth:      authManager,
		branding:  brandingManager,
		community: communityService,
		store:     authManager.Store(),
	}
}

func (manager *ApiManagerCtx) Mount(r chi.Router) {
	r.Use(manager.auth.Middleware)
	r.Use(manager.auth.CSRF)

	//
	// public
	//

	r.Get("/auth/status", manager.authStatus)
	r.Post("/auth/login", manager.authLogin)
	r.Post("/auth/logout", manager.authLogout)
	r.Post("/auth/register", manager.authRegister)
	r.Post("/auth/setup", manager.authSetup)

	r.Get("/branding", manager.brandingGet)
	r.Get("/branding/assets/{name}", manager.brandingAsset)

	r.Route("/public/rooms", func(r chi.Router) {
		r.Get("/", manager.publicRooms)
		r.Get("/{roomName}", manager.publicRoom)
		r.Post("/{roomName}/join", manager.publicRoomJoin)
		r.Get("/{roomName}/thumbnail.jpg", manager.publicRoomThumbnail)
	})

	//
	// signed in users
	//

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireUser)

		r.Put("/account", manager.accountUpdate)
		r.Post("/account/password", manager.accountPassword)
		r.Get("/account/sessions", manager.accountSessions)
		r.Delete("/account/sessions", manager.accountSessionsRevoke)

		r.Group(func(r chi.Router) {
			r.Use(manager.friendsEnabled)
			r.Get("/friends", manager.friendsList)
			r.Post("/friends", manager.friendsRequest)
			r.Post("/friends/{userId}/accept", manager.friendsAccept)
			r.Delete("/friends/{userId}", manager.friendsRemove)
			r.Get("/users/search", manager.usersSearch)
		})

		//
		// config
		//

		r.Get("/config/rooms", manager.configRooms)

		//
		// pull
		//

		r.Route("/pull", func(r chi.Router) {
			r.Use(manager.requirePullPermission)
			r.Get("/", manager.pullStatus)
			r.Get("/sse", manager.pullStatusSSE)
			r.Post("/", manager.pullStart)
			r.Delete("/", manager.pullStop)
		})

		//
		// rooms (owned rooms for users, all rooms for admins)
		//

		r.Get("/rooms", manager.roomsList)
		r.Post("/rooms", manager.roomCreate)

		// roomId is actually room name here
		r.Get("/rooms/{roomId}/by-name", manager.roomGetEntryByName)

		r.Route("/rooms/{roomId}", func(r chi.Router) {
			r.Use(manager.requireRoomManager)

			r.Get("/", manager.roomGetEntry)

			r.Get("/settings", manager.roomGetSettings)
			r.Get("/stats", manager.roomGetStats)

			r.Get("/meta", manager.roomMetaGet)
			r.Put("/meta", manager.roomMetaUpdate)
			r.Post("/invite", manager.roomInviteRegenerate)

			r.Delete("/", manager.roomRemove)
			r.Post("/start", manager.roomGenericAction("room.start", manager.rooms.Start))
			r.Post("/stop", manager.roomGenericAction("room.stop", manager.rooms.Stop))
			r.Post("/restart", manager.roomGenericAction("room.restart", manager.rooms.Restart))
			r.Post("/pause", manager.roomGenericAction("room.pause", manager.rooms.Pause))
			r.Post("/recreate", manager.roomRecreate)
		})

		//
		// events
		//

		r.Get("/events", manager.events)
	})

	//
	// admins
	//

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAdmin)

		r.Get("/docker-compose.yaml", manager.dockerCompose)

		r.Route("/admin", func(r chi.Router) {
			r.Get("/overview", manager.adminOverview)

			r.Get("/users", manager.adminUsersList)
			r.Post("/users", manager.adminUsersCreate)
			r.Put("/users/{userId}", manager.adminUsersUpdate)
			r.Delete("/users/{userId}", manager.adminUsersDelete)
			r.Get("/users/{userId}/sessions", manager.adminUserSessions)
			r.Delete("/users/{userId}/sessions", manager.adminUserSessionsRevoke)

			r.Get("/rooms", manager.adminRooms)

			r.Get("/policy", manager.adminPolicyGet)
			r.Put("/policy", manager.adminPolicySet)

			r.Get("/branding/defaults", manager.brandingDefaults)
			r.Put("/branding", manager.adminBrandingSet)
			r.Post("/branding/reset", manager.adminBrandingReset)
			r.Get("/branding/assets", manager.adminAssetsList)
			r.Post("/branding/assets/{name}", manager.adminAssetUpload)
			r.Delete("/branding/assets/{name}", manager.adminAssetDelete)

			r.Get("/audit", manager.adminAudit)
		})
	})
}
