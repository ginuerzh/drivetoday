// log
package controllers

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	//"strconv"
	"strings"
	"time"
)

const (
	redisStatVisitorPrefix     = "drivetoday:stat:visitors:"
	redisStatPvPrefix          = "drivetoday:stat:pv:"
	redisStatRegisterPrefix    = "drivetoday:stat:registers:"
	redisStatArticleViewPrefix = "drivetoday:stat:articles:view:"
	redisStatArticleReview     = "drivetoday:stat:articles:review"
	redisStatArticleThumb      = "drivetoday:stat:articles:thumb"
	redisUserMessagePrefix     = "drivetoday:user:msgs:"
	redisArticlePrefix         = "drivetoday:article:"
)

func dateString(time time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d", time.Year(), time.Month(), time.Day())
}

func redisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
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

type RedisLogger struct {
	pool *redis.Pool
}

func NewRedisLogger() *RedisLogger {
	return &RedisLogger{pool: redisPool()}
}

func (logger *RedisLogger) setsCount(key string, days int) []int64 {
	if days <= 0 {
		days = 1
	}

	t := time.Now()
	d, _ := time.ParseDuration("-24h")

	conn := logger.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("SCARD", key+dateString(t))
	for i := 1; i < days; i++ {
		t = t.Add(d)
		conn.Send("SCARD", key+dateString(t))
	}
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		log.Println(err)
		return nil
	}

	counts := make([]int64, len(values))
	for i, v := range values {
		counts[i], _ = v.(int64)
	}

	return counts
}

func (logger *RedisLogger) LogArticle(articleId string, article []byte) {
	d := time.Minute * 5
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Do("SETEX", redisArticlePrefix+articleId, int(d.Seconds()), article)
}

func (logger *RedisLogger) GetArticle(articleId string) []byte {
	conn := logger.pool.Get()
	defer conn.Close()

	s, err := redis.Bytes(conn.Do("GET", redisArticlePrefix+articleId))
	if err != nil {
		log.Println(err)
	}
	return s
}

func (logger *RedisLogger) LogUserMessages(userid string, msgs ...string) {
	args := redis.Args{}.Add(redisUserMessagePrefix + userid).AddFlat(msgs)
	conn := logger.pool.Get()
	defer conn.Close()
	conn.Do("LPUSH", args...)
}

func (logger *RedisLogger) MessageCount(userid string) int {
	conn := logger.pool.Get()
	defer conn.Close()
	count, err := redis.Int(conn.Do("LLEN", redisUserMessagePrefix+userid))
	if err != nil {
		log.Println(err)
	}
	return count
}

func (logger *RedisLogger) ClearMessages(userid string) {
	conn := logger.pool.Get()
	defer conn.Close()
	conn.Do("DEL", redisUserMessagePrefix+userid)
}

// log unique visitors per day
func (logger *RedisLogger) LogVisitor(ip string) {
	conn := logger.pool.Get()
	defer conn.Close()
	conn.Do("SADD", redisStatVisitorPrefix+dateString(time.Now()), ip)
}

func (logger *RedisLogger) VisitorsCount(days int) []int64 {
	return logger.setsCount(redisStatVisitorPrefix, days)
}

// log pv per day
func (logger *RedisLogger) LogPV(path string) {
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Do("ZINCRBY", redisStatPvPrefix+dateString(time.Now()), 1, path)
}

type KV struct {
	K string `json:"path"`
	V int64  `json:"count"`
}

func (logger *RedisLogger) PVs(dates ...string) map[string][]KV {
	if len(dates) == 0 {
		dates = []string{dateString(time.Now())}
	}

	pvs := make(map[string][]KV, len(dates))

	for _, date := range dates {
		pvs[date] = logger.PV(date)
	}

	return pvs
}

func (logger *RedisLogger) PV(date string) []KV {
	if len(date) == 0 {
		return nil
	}

	conn := logger.pool.Get()
	defer conn.Close()

	count, _ := redis.Int(conn.Do("ZCARD", redisStatPvPrefix+date))
	values, err := redis.Values(conn.Do("ZREVRANGE", redisStatPvPrefix+date, 0, count, "WITHSCORES"))

	if err != nil {
		log.Println(err)
		return nil
	}

	var pvs []KV

	if err := redis.ScanSlice(values, &pvs); err != nil {
		log.Println(err)
		return nil
	}
	return pvs
}

// log register users per day
func (logger *RedisLogger) LogRegister(userid string) {
	conn := logger.pool.Get()
	defer conn.Close()
	conn.Do("SADD", redisStatRegisterPrefix+dateString(time.Now()), userid)
}

func (logger *RedisLogger) RegisterCount(days int) []int64 {
	return logger.setsCount(redisStatRegisterPrefix, days)
}

func (logger *RedisLogger) LogArticleView(articleId string) {
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Do("ZINCRBY", redisStatArticleViewPrefix+dateString(time.Now()), 1, articleId)
}

func (logger *RedisLogger) ArticleTopView(days, max int) []string {
	if days <= 0 {
		days = 1
	}
	if max <= 0 {
		max = 3
	}

	t := time.Now()
	d, _ := time.ParseDuration("-24h")

	keys := make([]string, days)
	keys[0] = redisStatArticleViewPrefix + dateString(t)
	for i := 1; i < days; i++ {
		t = t.Add(d)
		keys[i] = redisStatArticleViewPrefix + dateString(t)
	}

	args := redis.Args{}.Add(redisStatArticleViewPrefix + "out").Add(days).AddFlat(keys)
	//log.Println(args)
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("ZUNIONSTORE", args...)
	conn.Send("ZREVRANGE", redisStatArticleViewPrefix+"out", 0, max, "WITHSCORES")
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil || len(values) < 2 {
		log.Println(err)
		return nil
	}

	var tops []KV
	s, _ := values[1].([]interface{})

	if err := redis.ScanSlice(s, &tops); err != nil {
		log.Println(err)
		return nil
	}

	articles := make([]string, len(tops))
	for i, _ := range tops {
		articles[i] = tops[i].K
	}

	return articles
}

func (logger *RedisLogger) LogArticleReview(articleId string) {
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Do("ZINCRBY", redisStatArticleReview, 1, articleId)
}

func (logger *RedisLogger) ArticleReviewCount(articleId string) int {
	conn := logger.pool.Get()
	defer conn.Close()

	count, err := redis.Int(conn.Do("ZSCORE", redisStatArticleReview, articleId))
	if err != nil {
		//log.Println(err)
	}
	return count
}

func (logger *RedisLogger) ArticleTopReview(max int) []string {
	if max <= 0 {
		max = 1
	}
	conn := logger.pool.Get()
	defer conn.Close()

	articles, err := redis.Strings(conn.Do("ZREVRANGE", redisStatArticleReview, 0, max))
	if err != nil {
		log.Println(err)
		return nil
	}

	return articles
}

func (logger *RedisLogger) LogArticleThumb(articleId string, thumb bool) {
	inc := 1
	if !thumb {
		inc = -1
	}
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Do("ZINCRBY", redisStatArticleThumb, inc, articleId)
}

func (logger *RedisLogger) ArticleThumbCount(articleId string) int {
	conn := logger.pool.Get()
	defer conn.Close()

	count, err := redis.Int(conn.Do("ZSCORE", redisStatArticleThumb, articleId))
	if err != nil {
		//log.Println(err)
	}
	return count
}

func (logger *RedisLogger) ArticleTopThumb(max int) []string {
	if max <= 0 {
		max = 1
	}
	conn := logger.pool.Get()
	defer conn.Close()

	articles, err := redis.Strings(conn.Do("ZREVRANGE", redisStatArticleThumb, 0, max))
	if err != nil {
		log.Println(err)
		return nil
	}

	return articles
}

func LogRequestHandler(request *http.Request, redisLogger *RedisLogger) {
	s := strings.Split(request.RemoteAddr, ":")

	if len(s) > 0 {
		redisLogger.LogVisitor(s[0])
	}
	redisLogger.LogPV(request.URL.Path)
}
