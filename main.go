// main
package main

import (
	"github.com/codegangsta/martini"
	"github.com/garyburd/redigo/redis"
	"github.com/ginuerzh/drivetoday/controllers"
	"log"
	"net/http"
	"os"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func redisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "localhost:6379")
			if err != nil {
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

func main() {
	m := martini.Classic()
	m.Map(log.New(os.Stdout, "[martini] ", log.LstdFlags))
	m.Map(redisPool())

	controllers.BindUserApi(m)
	controllers.BindArticleApi(m)
	controllers.BindReviewApi(m)
	controllers.BindFileApi(m)

	http.ListenAndServe(":8080", m)
}
