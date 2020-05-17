package http

import (
	"slicerapi/internal/http/ws"
	"slicerapi/internal/util"

	"github.com/gin-gonic/gin"
)

// Start starts the HTTP server.
func Start() {
	r := gin.New()
	register(r)
	util.Chk(r.Run(util.Config.HTTP.Address))
}

// register registers all routes and middleware.
func register(r *gin.Engine) {
	authMiddlewareFunc := authMiddleware.MiddlewareFunc()

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handleRegister)
			auth.POST("/login", authMiddleware.LoginHandler)
			auth.GET("/refresh", authMiddleware.RefreshHandler)
		}

		channel := v1.Group("/channel/:channel")
		channel.Use(authMiddlewareFunc)
		{
			channel.POST("/message", handleAddMessage)
		}

		websocket := v1.Group("/ws")
		websocket.Use(authMiddlewareFunc)
		{
			ws.NewController(true)
			go ws.C.Run()

			websocket.GET("", func(c *gin.Context) {
				ws.Handle(c)
			})
		}
	}
}
