package handler

import (
	"ares-operator/conf"

	"github.com/gin-gonic/gin"
)

// InitRouter: 初始化API路由
func InitRouter(config conf.ObserverConfig) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	apiGroup := router.Group("cache")
	apiGroup.GET("/ping", Ping)
	apiGroup.GET("/keys/:key", GetJobCache)
	return router
}
