// review
package controllers

import (
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"labix.org/v2/mgo/bson"
	//"log"
	"net/http"
	"strings"
	"time"
)

const (
	ReviewListV1Uri     = "/1/review/list"
	ReviewNewV1Uri      = "/1/review/new"
	ReviewSetThumbV1Uri = "/1/review/thumb"
	ReviewThumbedV1Uri  = "/1/review/is_thumbed"
)

func BindReviewApi(m *martini.ClassicMartini) {
	m.Get(ReviewListV1Uri, binding.Form(reviewListForm{}), ErrorHandler, reviewListHandler)
	m.Post(ReviewNewV1Uri, binding.Json(newReviewForm{}), ErrorHandler, newReviewHandler)

	m.Post(ReviewSetThumbV1Uri, binding.Json(reviewThumbForm{}), ErrorHandler, reviewSetThumbHandler)
	m.Get(ReviewThumbedV1Uri, binding.Form(reviewThumbForm{}), ErrorHandler, checkReviewThumbHandler)
}

type reviewListForm struct {
	ArticleId   string `form:"article_id" json:"article_id"`
	PageNumber  int    `form:"page_number" json:"page_number"`
	AccessToken string `form:"access_token" json:"access_token"`
}

func (form *reviewListForm) Validate(e *binding.Errors, req *http.Request) {
	if len(form.ArticleId) > 0 && !bson.IsObjectIdHex(form.ArticleId) {
		e.Fields["id"] = "invalid article id"
	}
}

type reviewJsonStruct struct {
	Id        string `json:"review_id"`
	ArticleId string `json:"article_id"`
	Userid    string `json:"review_author"`
	Content   string `json:"message"`
	Thumbs    int    `json:"thumb_count"`
	Ctime     string `json:"time"`
}

func reviewListHandler(request *http.Request, resp http.ResponseWriter, form reviewListForm, redis *RedisLogger) {
	var total, err int
	var reviews []models.Review

	if len(form.ArticleId) > 0 {
		article := models.Article{}
		article.Id = bson.ObjectIdHex(form.ArticleId)
		total, reviews, err = article.Reviews(DefaultPageSize*form.PageNumber, DefaultPageSize)
	}

	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStructs := make([]reviewJsonStruct, len(reviews))
	for i, _ := range reviews {
		jsonStructs[i].Id = reviews[i].Id.Hex()
		jsonStructs[i].ArticleId = reviews[i].ArticleId
		jsonStructs[i].Userid = reviews[i].Userid
		jsonStructs[i].Content = reviews[i].Content
		jsonStructs[i].Thumbs = len(reviews[i].Thumbs)
		jsonStructs[i].Ctime = reviews[i].Ctime.Format(TimeFormat)
	}

	respData := make(map[string]interface{})
	respData["page_number"] = form.PageNumber
	respData["page_more"] = DefaultPageSize*(form.PageNumber+1) < total
	respData["total"] = total
	respData["reviews"] = jsonStructs

	writeResponse(request.RequestURI, resp, respData, err)
}

type newReviewForm struct {
	ArticleId   string `form:"article_id" json:"article_id"`
	Content     string `form:"contents" json:"contents"`
	AccessToken string `form:"access_token" json:"access_token"`
}

func (form *newReviewForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ArticleId) {
		e.Fields["id"] = "invalid article id"
		return
	}
}

func findMentions(review string) []string {
	mentions := []string{}

	if !strings.Contains(review, "@") {
		return mentions
	}

	s := strings.Split(review, " ")
	for i, _ := range s {
		if strings.HasPrefix(s[i], "@") && strings.Count(s[i], "@") == 1 {
			mentions = append(mentions, s[i])
		}
	}

	return mentions
}

func newReviewHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form newReviewForm) {
	var review models.Review

	conn := redis.Conn()
	defer conn.Close()

	userid := redis.OnlineUser(conn, form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	review.ArticleId = form.ArticleId
	review.Userid = userid
	review.Content = form.Content
	review.Ctime = time.Now()

	err := review.Save()
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStruct := &reviewJsonStruct{}
	jsonStruct.Id = review.Id.Hex()
	jsonStruct.ArticleId = review.ArticleId
	jsonStruct.Userid = review.Userid
	jsonStruct.Content = review.Content
	jsonStruct.Thumbs = len(review.Thumbs)
	jsonStruct.Ctime = review.Ctime.Format(TimeFormat)

	writeResponse(request.RequestURI, resp, jsonStruct, err)

	user := models.User{Userid: userid}
	user.RateArticle(form.ArticleId, ReviewRate, false)
	redis.LogArticleReview(conn, userid, form.ArticleId)

	for _, mention := range findMentions(review.Content) {
		nickname := strings.TrimLeft(mention, "@")
		user := models.User{}
		if find, _ := user.FindByNickname(nickname); !find {
			continue
		}

		event := models.Event{}
		event.Type = "review"
		event.Ctime = time.Now()
		event.ArticleId = form.ArticleId
		event.User = userid
		event.Owner = user.Userid
		//event.Read = false
		event.Message = nickname + "在评论中提到了你！"

		if err := event.Save(); err == errors.NoError {
			redis.LogUserMessages(conn, event.Owner, event.Json())
		}
	}
}

type reviewThumbForm struct {
	ArticleId   string `form:"article_id" json:"article_id"`
	ReviewId    string `form:"review_id" json:"review_id" binding:"required"`
	Status      bool   `form:"thumb_status" json:"thumb_status"`
	AccessToken string `form:"access_token" json:"access_token" binding:"required"`
}

func (form *reviewThumbForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ReviewId) {
		e.Fields["id"] = "invalid article id"
		return
	}
}

func reviewSetThumbHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form reviewThumbForm) {
	var review models.Review
	var user models.User

	conn := redis.Conn()
	defer conn.Close()

	userid := redis.OnlineUser(conn, form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	if find, err := user.FindByUserId(userid); !find {
		if err == errors.NoError {
			err = errors.NotFoundError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}
	if user.Role == UserTypeGuest {
		user.Nickname = "匿名用户"
	}

	if find, err := review.FindById(form.ReviewId); !find {
		if err == errors.NoError {
			err = errors.NotFoundError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	err := review.SetThumb(userid, form.Status)

	writeResponse(request.RequestURI, resp, nil, err)

	event := models.Event{}
	event.Type = "thumb"
	event.Ctime = time.Now()
	event.ArticleId = form.ArticleId
	if user.Role != UserTypeGuest {
		event.User = userid
	}
	event.Owner = review.Userid
	//event.Read = false
	event.Message = user.Nickname + "赞了你的评论!"
	if err := event.Save(); err == errors.NoError {
		redis.LogUserMessages(conn, event.Owner, event.Json())
	}
}

func checkReviewThumbHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form reviewThumbForm) {
	var review models.Review

	userid := redis.OnlineUser(nil, form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	review.Id = bson.ObjectIdHex(form.ReviewId)
	thumbed, err := review.IsThumbed(userid)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
	}

	respData := make(map[string]bool, 1)
	respData["is_thumbed"] = thumbed
	writeResponse(request.RequestURI, resp, respData, err)
}
