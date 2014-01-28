// redis
package controllers

import (
	"github.com/codegangsta/martini"
	"github.com/garyburd/redigo/redis"
	"github.com/ginuerzh/drivetoday/models"
	"net/http"
	"strings"
)

func RedisLoggerHandler(request *http.Request, c martini.Context, pool *redis.Pool) {
	logger := models.NewRedisLogger(pool.Get())
	defer logger.Close()

	// log request
	s := strings.Split(request.RemoteAddr, ":")
	if len(s) > 0 {
		logger.LogVisitor(s[0])
	}
	logger.LogPV(request.URL.Path)

	c.Map(logger)
	c.Next()
}
