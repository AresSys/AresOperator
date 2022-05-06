package handler

import (
	"ares-operator/observer/server"

	"github.com/gin-gonic/gin"

	"ares-operator/cache"
	"ares-operator/observer"
)

// @Summary Ping this server
// @Tags Cache
// @Accept json
// @Produce string
// @Success 200 {string} string
// @Router /cache/ping [get]
func Ping(c *gin.Context) {
	server.Success(c, "pong")
}

// @Summary Get JobCache
// @Tags Cache
// @Accept json
// @Produce json
// @Param key path string true "key of job"
// @Success 200 {object} cache.JobCache
// @Router /cache/keys/{key} [get]
func GetJobCache(c *gin.Context) {
	var param JobKeyParam
	if err := c.ShouldBindUri(&param); err != nil {
		server.Failure(c, observer.InvalidParam, err)
		return
	}
	ns, name, err := cache.KeyToNamespacedName(param.Key)
	if err != nil {
		server.Failure(c, observer.InvalidParam, err)
		return
	}
	jobCache, err := observer.GetJobCache(ns, name)
	if err != nil {
		observer.HandleError(c, err)
		return
	}
	server.Success(c, jobCache)
}
