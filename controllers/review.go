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
	ArticleId   string      `form:"article_id" json:"article_id"`
	PageNumber  int         `form:"page_number" json:"page_number"`
	AccessToken string      `form:"access_token" json:"access_token"`
	User        models.User `form:"-"`
}

func (form *reviewListForm) Validate(e *binding.Errors, req *http.Request) {
	if len(form.ArticleId) > 0 {
		if !bson.IsObjectIdHex(form.ArticleId) {
			e.Fields["id"] = "invalid article id"
		}
	} else {
		form.User = userAuth(form.AccessToken, e)
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

func reviewListHandler(request *http.Request, resp http.ResponseWriter, form reviewListForm) {
	var total, err int
	var reviews []models.Review

	if len(form.ArticleId) > 0 {
		article := models.Article{}
		article.Id = bson.ObjectIdHex(form.ArticleId)
		total, reviews, err = article.Reviews(DefaultPageSize*form.PageNumber, DefaultPageSize)
	} else {
		total, reviews, err = form.User.Reviews(DefaultPageSize*form.PageNumber, DefaultPageSize)
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
	ArticleId   string      `form:"article_id" json:"article_id"`
	Content     string      `form:"contents" json:"contents"`
	AccessToken string      `form:"access_token" json:"access_token"`
	User        models.User `form"-" json:"-"`
}

func (form *newReviewForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ArticleId) {
		e.Fields["id"] = "invalid article id"
		return
	}

	form.User = userAuth(form.AccessToken, e)
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

	review.ArticleId = form.ArticleId
	review.Userid = form.User.Userid
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

	redis.LogArticleReview(form.ArticleId)

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
		event.User = form.User.Userid
		event.Owner = user.Userid
		event.Read = false
		event.Message = nickname + "在一条评论中提到了你！"

		if err := event.Save(); err == errors.NoError {
			redis.LogUserMessages(event.Owner, event.Json())
		}
	}
}

type reviewThumbForm struct {
	ArticleId   string      `form:"article_id" json:"article_id" binding:"required"`
	ReviewId    string      `form:"review_id" json:"review_id" binding:"required"`
	Status      bool        `form:"thumb_status" json:"thumb_status"`
	AccessToken string      `form:"access_token" json:"access_token" binding:"required"`
	User        models.User `form"-" json:"-"`
}

func (form *reviewThumbForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ReviewId) {
		e.Fields["id"] = "invalid article id"
		return
	}
	form.User = userAuth(form.AccessToken, e)
}

func reviewSetThumbHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form reviewThumbForm) {
	var review models.Review

	if find, err := review.FindById(form.ReviewId); !find {
		if err == errors.NoError {
			err = errors.NotExistsError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	err := review.SetThumb(form.User.Userid, form.Status)

	writeResponse(request.RequestURI, resp, nil, err)

	event := models.Event{}
	event.Type = "thumb"
	event.Ctime = time.Now()
	event.ArticleId = form.ArticleId
	event.User = form.User.Userid
	event.Owner = review.Userid
	event.Read = false
	event.Message = form.User.Nickname + "赞了你的评论!"
	if err := event.Save(); err == errors.NoError {
		redis.LogUserMessages(event.Owner, event.Json())
	}

	/*
		conn := pool.Get()
		defer conn.Close()
		if err != errors.NoError && form.Status {
			conn.Send("RPUSH", "drivetoday:user:review:thumbs:"+review.Userid, form.User.Nickname)
		}
		if _, err := conn.Do("EXEC"); err != nil {
			log.Println(err)
		}
	*/
}

func checkReviewThumbHandler(request *http.Request, resp http.ResponseWriter, form reviewThumbForm) {
	var review models.Review
	review.Id = bson.ObjectIdHex(form.ReviewId)
	thumbed, err := review.IsThumbed(form.User.Userid)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
	}

	respData := make(map[string]bool, 1)
	respData["is_thumbed"] = thumbed
	writeResponse(request.RequestURI, resp, respData, err)
}
