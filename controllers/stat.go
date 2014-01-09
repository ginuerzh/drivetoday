// stat
package controllers

import (
	"github.com/codegangsta/martini"
	"github.com/ginuerzh/drivetoday/errors"
	"net/http"
	"time"
)

const (
	ServerStatV1Uri = "/1/stat"
)

func BindStatApi(m *martini.ClassicMartini) {
	m.Get(ServerStatV1Uri, serverStatHandler)
}

func serverStatHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger) {
	respData := make(map[string]interface{})
	respData["visitors"] = redis.VisitorsCount(3)
	respData["pv"] = redis.PV(dateString(time.Now()))
	respData["registers"] = redis.RegisterCount(3)

	respData["top_views"] = redis.ArticleTopView(3, 3)
	respData["top_reviews"] = redis.ArticleTopReview(3)
	respData["top_thumbs"] = redis.ArticleTopThumb(3)

	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}
