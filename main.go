// main
package main

import (
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/ginuerzh/drivetoday/controllers"
	//"github.com/martini-contrib/gzip"
	"gopkg.in/go-martini/martini.v1"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	staticDir string
	port      int
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&staticDir, "static", "public", "static files directory")
	flag.IntVar(&port, "port", 8080, "port on listen")
	flag.Parse()
}

func classic() *martini.ClassicMartini {
	r := martini.NewRouter()
	m := martini.New()
	//m.Use(gzip.All())
	//m.Use(martini.Logger())
	m.Use(controllers.RedisLoggerHandler)
	m.Use(martini.Recovery())
	m.Use(martini.Static(staticDir))
	m.Action(r.Handle)
	return &martini.ClassicMartini{m, r}
}

func main() {
	m := classic()
	m.Map(log.New(os.Stdout, "[martini] ", log.LstdFlags))
	m.Map(redisPool())
	//m.Map(controllers.NewRedisLogger())

	controllers.BindUserApi(m)
	controllers.BindArticleApi(m)
	controllers.BindReviewApi(m)
	controllers.BindFileApi(m)
	controllers.BindEventApi(m)
	controllers.BindStatApi(m)

	//m.Run()
	http.ListenAndServe(":"+strconv.Itoa(port), m)
}

func redisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "localhost:6379")
			if err != nil {
				log.Println(err)
				return nil, err
			}
			/*
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			*/
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
