package api

import (
	"time"

	"github.com/mmanjoura/clean-bet-backend/pkg/api/racing"
	"github.com/mmanjoura/clean-bet-backend/pkg/auth"
	"github.com/mmanjoura/clean-bet-backend/pkg/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	docs "github.com/mmanjoura/clean-bet-backend/cmd/docs"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitRouter initializes the routes for the API
func InitRouter() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Logger())
	r.Use(middleware.Cors())
	r.Use(middleware.RateLimiter(rate.Every(1*time.Minute), 600)) // 60 requests per minute
	docs.SwaggerInfo.BasePath = "/api/v1"

	v1 := r.Group("/api/v1")
	{
		v1.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
		// Auth routes
		v1.POST("/auth/login", auth.LoginHandler)
		v1.POST("/auth/register", auth.RegisterHandler)
		v1.POST("/auth/logout", auth.Logout)

		// meeting routes
		v1.GET("/racing/events", racing.GetEvents)
		v1.GET("/racing/selections", racing.GetSelections)
		v1.POST("/racing/meetings", racing.GetMeetings)
		v1.POST("/racing/analysis", racing.DoAnalysis)
		v1.POST("/racing/results", racing.GetResults)
		v1.POST("/racing/forms", racing.GetForms)
		v1.POST("/racing/predictions", racing.GetPredictions)
	}

	return r
}
