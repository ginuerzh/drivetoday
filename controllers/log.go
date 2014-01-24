// log
package controllers

import (
	//"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	redisStatVisitorPrefix     = "drivetoday:stat:visitors:"       // set per day
	redisStatPvPrefix          = "drivetoday:stat:pv:"             // sorted set per day
	redisStatRegisterPrefix    = "drivetoday:stat:registers:"      // set per day
	redisStatArticleViewPrefix = "drivetoday:stat:articles:view:"  // sorted set per day
	redisStatArticleView       = "drivetoday:stat:articles:view"   // sorted set
	redisStatArticleReview     = "drivetoday:stat:articles:review" // sorted set
	redisStatArticleThumb      = "drivetoday:stat:articles:thumb"  // sorted set

	redisArticleCachePrefix   = "drivetoday:article:cache:"   // string per article
	redisArticleViewPrefix    = "drivetoday:article:view:"    // list per article
	redisArticleThumbPrefix   = "drivetoday:article:thumb:"   // list per article
	redisArticleReviewPrefix  = "drivetoday:article:review:"  // list per article
	redisArticleRelatedPrefix = "drivetoday:article:related:" // sorted set per article

	redisUserMessagePrefix    = "drivetoday:user:msgs:"     // list per user
	redisUserOnlinesPrefix    = "drivetoday:user:onlines:"  // set per half an hour
	redisUserOnlineUserPrefix = "drivetoday:user:online:"   // string per user
	redisUserGuest            = "drivetoday:user:guest"     // hashes for all guests
	redisUserArticlePrefix    = "drivetoday:user:articles:" // sorted set per user
)

const (
	onlineUserExpire = 15 * 60 // 15m online user timeout
	onlinesExpire    = 60 * 60 // 60m online set timeout
)

func onlineTimeString() string {
	now := time.Now()
	min := now.Minute()
	if min < 30 {
		now = now.Add(time.Duration(0-min) * time.Minute)
	} else {
		now = now.Add(time.Duration(30-min) * time.Minute)
	}
	return now.Format("200601021504")
}

func dateString(t time.Time) string {
	return t.Format("2006-01-02")
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

func (logger *RedisLogger) Conn() redis.Conn {
	return logger.pool.Get()
}

// log register users per day
func (logger *RedisLogger) LogRegister(conn redis.Conn, userid string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}
	conn.Do("SADD", redisStatRegisterPrefix+dateString(time.Now()), userid)
}

func (logger *RedisLogger) RegisterCount(conn redis.Conn, days int) []int64 {
	return logger.setsCount(conn, redisStatRegisterPrefix, days)
}

func (logger *RedisLogger) OnlineUser(conn redis.Conn, accessToken string) string {
	if len(accessToken) == 0 {
		return ""
	}

	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	var userid string

	if strings.HasPrefix(accessToken, GuestUserPrefix) {
		userid, _ = redis.String(conn.Do("HGET", redisUserGuest, accessToken))
	} else {
		userid, _ = redis.String(conn.Do("GET", redisUserOnlineUserPrefix+accessToken))
	}

	logger.LogOnlineUser(conn, accessToken, userid)

	return userid
}

func (logger *RedisLogger) LogOnlineUser(conn redis.Conn, accessToken, userid string) {
	if len(accessToken) == 0 || len(userid) == 0 {
		return
	}
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("MULTI")
	if !strings.HasPrefix(accessToken, GuestUserPrefix) {
		conn.Send("SETEX", redisUserOnlineUserPrefix+accessToken, onlineUserExpire, userid)
	} else {
		conn.Send("HSETNX", redisUserGuest, accessToken, userid)
	}
	timeStr := onlineTimeString()
	conn.Send("SADD", redisUserOnlinesPrefix+timeStr, userid)
	conn.Send("EXPIRE", redisUserOnlinesPrefix+timeStr, onlinesExpire)
	conn.Do("EXEC")
}

func (logger *RedisLogger) DelOnlineUser(conn redis.Conn, accessToken string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("DEL", redisUserOnlineUserPrefix+accessToken)
}

func (logger *RedisLogger) Onlines(conn redis.Conn) int {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	count, _ := redis.Int(conn.Do("SCARD", redisUserOnlinesPrefix+onlineTimeString()))
	return count
}

func (logger *RedisLogger) setsCount(conn redis.Conn, key string, days int) []int64 {
	if days <= 0 {
		days = 1
	}

	t := time.Now()
	d, _ := time.ParseDuration("-24h")

	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

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

/*
func (logger *RedisLogger) LogUserArticle(userid, article string, rate int) {
	conn := logger.pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	if (rate | AccessRate) != 0 {
		conn.Send("ZADD", redisUserArticlePrefix+userid, rate, article)
	}
	if (rate | ThumbRate) != 0 {
		conn.Send("ZADD", redisUserArticlePrefix+userid, rate, article)
	}
}
*/
func (logger *RedisLogger) UserArticleRate(conn redis.Conn, userid string, articles ...string) []int {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	rates := make([]int, len(articles))
	conn.Send("MULTI")
	for _, article := range articles {
		conn.Send("ZSCORE", redisUserArticlePrefix+userid, article)
	}
	if values, err := redis.Strings(conn.Do("EXEC")); err == nil {
		for i, v := range values {
			rates[i], _ = strconv.Atoi(v)
		}
	}

	return rates
}

func (logger *RedisLogger) LogArticleCache(conn redis.Conn, articleId string, article []byte) {
	d := time.Minute * 5

	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Do("SETEX", redisArticleCachePrefix+articleId, int(d.Seconds()), article)
}

func (logger *RedisLogger) GetArticleCache(conn redis.Conn, articleId string) []byte {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	s, err := redis.Bytes(conn.Do("GET", redisArticleCachePrefix+articleId))
	if err != nil {
		//log.Println(err)
	}
	return s
}

func (logger *RedisLogger) LogUserMessages(conn redis.Conn, userid string, msgs ...string) {
	args := redis.Args{}.Add(redisUserMessagePrefix + userid).AddFlat(msgs)
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}
	conn.Do("LPUSH", args...)
}

func (logger *RedisLogger) MessageCount(conn redis.Conn, userid string) int {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	count, err := redis.Int(conn.Do("LLEN", redisUserMessagePrefix+userid))
	if err != nil {
		log.Println(err)
	}
	return count
}

func (logger *RedisLogger) ClearMessages(conn redis.Conn, userid string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Do("DEL", redisUserMessagePrefix+userid)
}

// log unique visitors per day
func (logger *RedisLogger) LogVisitor(conn redis.Conn, ip string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Do("SADD", redisStatVisitorPrefix+dateString(time.Now()), ip)
}

func (logger *RedisLogger) VisitorsCount(conn redis.Conn, days int) []int64 {
	return logger.setsCount(conn, redisStatVisitorPrefix, days)
}

// log pv per day
func (logger *RedisLogger) LogPV(conn redis.Conn, path string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Do("ZINCRBY", redisStatPvPrefix+dateString(time.Now()), 1, path)
}

type KV struct {
	K string `json:"path"`
	V int64  `json:"count"`
}

func (logger *RedisLogger) PVs(conn redis.Conn, dates ...string) map[string][]KV {
	if len(dates) == 0 {
		dates = []string{dateString(time.Now())}
	}

	pvs := make(map[string][]KV, len(dates))

	for _, date := range dates {
		pvs[date] = logger.PV(conn, date)
	}

	return pvs
}

func (logger *RedisLogger) PV(conn redis.Conn, date string) []KV {
	if len(date) == 0 {
		return nil
	}

	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

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

func (logger *RedisLogger) RelatedArticles(conn redis.Conn, article string, max int) []string {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	members, err := redis.Strings(conn.Do("SMEMBERS", redisArticleViewPrefix+article))
	if err != nil {
		log.Println(err)
		return nil
	}
	//log.Println(members)
	keys := make([]string, len(members))
	for i, _ := range members {
		keys[i] = redisUserArticlePrefix + members[i]
	}
	args := redis.Args{}.Add(redisArticleRelatedPrefix + article).Add(len(members)).AddFlat(keys)
	conn.Send("MULTI")
	conn.Send("ZUNIONSTORE", args...)
	conn.Send("ZREVRANGE", redisArticleRelatedPrefix+article, 0, max)
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		log.Println(err)
		return nil
	}

	//log.Println(values)
	s, ok := values[1].([]interface{})
	if !ok {
		return nil
	}

	var articles []string
	for i, _ := range s {
		bs, ok := s[i].([]byte)
		if !ok {
			log.Println(string(bs), "is not string")
		}
		id := string(bs)
		if len(id) > 0 && id != article {
			articles = append(articles, id)
		}

		if len(articles) == max {
			break
		}
	}
	return articles
}

func (logger *RedisLogger) ViewedArticles(conn redis.Conn, userid string) []string {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	count, _ := redis.Int(conn.Do("ZCARD", redisUserArticlePrefix+userid))
	values, err := redis.Strings(conn.Do("ZRANGE", redisUserArticlePrefix+userid, 0, count))

	if err != nil {
		log.Println(err)
		return nil
	}

	return values
}

func (logger *RedisLogger) ArticleCount(conn redis.Conn, articleId string) (view, thumb, review int64) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("MULTI")
	conn.Send("ZSCORE", redisStatArticleView, articleId)
	//conn.Send(conn.Do("SCARD", redisArticleViewPrefix+articleId))
	conn.Send("ZSCORE", redisStatArticleThumb, articleId)
	conn.Send("ZSCORE", redisStatArticleReview, articleId)
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
		log.Println(err)
		return
	}

	var counts []struct {
		Count int64
	}

	if err := redis.ScanSlice(values, &counts); err != nil {
		log.Println(err)
		return
	}

	view = counts[0].Count
	thumb = counts[1].Count
	review = counts[2].Count

	//	log.Println(view, thumb, review)

	return
}

func (logger *RedisLogger) LogArticleView(conn redis.Conn, articleId string, userid string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}
	//log.Println("log article view", articleId, userid)
	conn.Send("MULTI")
	conn.Send("ZINCRBY", redisStatArticleViewPrefix+dateString(time.Now()), 1, articleId)
	conn.Send("ZINCRBY", redisStatArticleView, 1, articleId)
	conn.Send("SADD", redisArticleViewPrefix+articleId, userid)
	conn.Send("ZADD", redisUserArticlePrefix+userid, AccessRate, articleId)
	conn.Do("EXEC")
}

func (logger *RedisLogger) ArticleView(conn redis.Conn, userid string, articles ...string) []bool {
	if len(userid) == 0 {
		return nil
	}

	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("MULTI")
	for _, article := range articles {
		conn.Send("SISMEMBER", redisArticleViewPrefix+article, userid)
	}
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil || len(values) != len(articles) {
		log.Println(err)
		return nil
	}

	views := make([]bool, len(articles))
	for i, v := range values {
		if b, ok := v.(int64); ok && b != 0 {
			views[i] = true
		}
	}
	return views
}

func (logger *RedisLogger) ArticleTopView(conn redis.Conn, days, max int) []string {
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
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("MULTI")
	conn.Send("ZUNIONSTORE", args...)
	conn.Send("ZREVRANGE", redisStatArticleViewPrefix+"out", 0, max, "WITHSCORES")
	values, err := redis.Values(conn.Do("EXEC"))
	if err != nil {
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

func (logger *RedisLogger) LogArticleReview(conn redis.Conn, userid, articleId string) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	conn.Send("MULTI")
	conn.Send("ZINCRBY", redisStatArticleReview, 1, articleId)
	conn.Send("SADD", redisArticleReviewPrefix+articleId, userid)
	conn.Send("ZADD", redisUserArticlePrefix+userid, ReviewRate, articleId)
	conn.Do("EXEC")
}

func (logger *RedisLogger) ArticleReviewCount(conn redis.Conn, articleId string) (count int) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	count, _ = redis.Int(conn.Do("ZSCORE", redisStatArticleReview, articleId))
	return
}

func (logger *RedisLogger) ArticleTopReview(conn redis.Conn, max int) []string {
	if max <= 0 {
		max = 1
	}
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	articles, err := redis.Strings(conn.Do("ZREVRANGE", redisStatArticleReview, 0, max))
	if err != nil {
		log.Println(err)
		return nil
	}

	return articles
}

func (logger *RedisLogger) LogArticleThumb(conn redis.Conn, userid, articleId string, thumb bool) {
	inc := 1
	if !thumb {
		inc = -1
	}
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}
	//log.Println("log article thumb", userid, articleId, thumb)
	conn.Send("MULTI")
	conn.Send("ZINCRBY", redisStatArticleThumb, inc, articleId)
	if thumb {
		conn.Send("SADD", redisArticleThumbPrefix+articleId, userid)
		conn.Send("ZADD", redisUserArticlePrefix+userid, ThumbRate, articleId)
	} else {
		conn.Send("SREM", redisArticleThumbPrefix+articleId, userid)
	}
	conn.Do("EXEC")
}

func (logger *RedisLogger) ArticleThumbed(conn redis.Conn, userid, articleId string) (b bool) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	b, _ = redis.Bool(conn.Do("SISMEMBER", redisArticleThumbPrefix+articleId, userid))
	return
}

func (logger *RedisLogger) ArticleThumbCount(conn redis.Conn, articleId string) (count int) {
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	count, _ = redis.Int(conn.Do("SCARD", redisArticleThumbPrefix+articleId))
	return
}

func (logger *RedisLogger) ArticleTopThumb(conn redis.Conn, max int) []string {
	if max <= 0 {
		max = 1
	}
	if conn == nil {
		conn = logger.pool.Get()
		defer conn.Close()
	}

	articles, err := redis.Strings(conn.Do("ZREVRANGE", redisStatArticleThumb, 0, max))
	if err != nil {
		log.Println(err)
		return nil
	}

	return articles
}

func LogRequestHandler(request *http.Request, redis *RedisLogger) {
	s := strings.Split(request.RemoteAddr, ":")

	conn := redis.Conn()
	defer conn.Close()

	if len(s) > 0 {
		redis.LogVisitor(conn, s[0])
	}
	redis.LogPV(conn, request.URL.Path)
}
