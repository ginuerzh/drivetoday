// main
package main

import (
	"github.com/codegangsta/martini"
	"github.com/ginuerzh/drivetoday/controllers"
	"log"
	"net/http"
	"os"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func classic() *martini.ClassicMartini {
	r := martini.NewRouter()
	m := martini.New()
	m.Use(controllers.LogRequestHandler)
	m.Use(martini.Logger())
	m.Use(martini.Recovery())
	m.Use(martini.Static("drivetodayweb"))
	m.Action(r.Handle)
	return &martini.ClassicMartini{m, r}
}

func main() {
	m := classic()
	m.Map(log.New(os.Stdout, "[martini] ", log.LstdFlags))
	m.Map(controllers.NewRedisLogger())

	controllers.BindUserApi(m)
	controllers.BindArticleApi(m)
	controllers.BindReviewApi(m)
	controllers.BindFileApi(m)
	controllers.BindEventApi(m)
	controllers.BindStatApi(m)

	//m.Run()
	http.ListenAndServe(":8080", m)
}
