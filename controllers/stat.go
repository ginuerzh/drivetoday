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
	conn := redis.Conn()
	defer conn.Close()

	respData := make(map[string]interface{})
	respData["visitors"] = redis.VisitorsCount(conn, 3)
	respData["pv"] = redis.PV(conn, dateString(time.Now()))
	respData["registers"] = redis.RegisterCount(conn, 3)

	respData["top_views"] = redis.ArticleTopView(conn, 3, 3)
	respData["top_reviews"] = redis.ArticleTopReview(conn, 3)
	respData["top_thumbs"] = redis.ArticleTopThumb(conn, 3)
	respData["onlines"] = redis.Onlines(conn)

	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}
