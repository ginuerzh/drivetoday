// article
package controllers

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"strings"
)

const (
	ArticleListV1Uri     = "/1/article/timelines"
	ArticleInfoV1Uri     = "/1/article/get"
	ArticleSetThumbV1Uri = "/1/article/thumb"
	ArticleThumbedV1Uri  = "/1/article/is_thumbed"
)

func BindArticleApi(m *martini.ClassicMartini) {
	m.Get(ArticleListV1Uri, binding.Form(articleListForm{}), ErrorHandler, articleListHandler)
	m.Get(ArticleInfoV1Uri, binding.Form(articleInfoForm{}), ErrorHandler, articleInfoHandler)

	m.Post(ArticleSetThumbV1Uri, binding.Json(articleThumbForm{}), ErrorHandler, articleSetThumbHandler)
	m.Get(ArticleThumbedV1Uri, binding.Form(articleThumbForm{}), ErrorHandler, checkArticleThumbHandler)
}

type articleListForm struct {
	PageNumber  int    `form:"page_number" json:"page_number"`
	AccessToken string `form:"access_token" json:"access_token"`
}

type contentObject struct {
	ContentType string `json:"seg_type"`
	ContentText string `json:"seg_content"`
	ImageUrl    string `json:"image_orig"`
}

type articleJsonStruct struct {
	Id      string          `json:"article_id"`
	Title   string          `json:"title"`
	Source  string          `json:"source"`
	Url     string          `json:"src_link"`
	PubTime string          `json:"publish_time"`
	Thumbs  int             `json:"thumb_count"`
	Reviews int             `json:"comment_count"`
	Image   string          `json:"first_image"`
	Content []contentObject `json:"content"`
}

func articleListHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleListForm) {
	total, articles, err := models.GetBriefArticles(DefaultPageSize*form.PageNumber, DefaultPageSize)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStructs := make([]articleJsonStruct, len(articles))
	for i, _ := range articles {
		jsonStructs[i].Id = articles[i].Id.Hex()
		jsonStructs[i].Title = articles[i].Title
		jsonStructs[i].Source = articles[i].Source
		jsonStructs[i].Url = articles[i].Url
		jsonStructs[i].PubTime = articles[i].PubTime.Format(TimeFormat)
		jsonStructs[i].Thumbs = redis.ArticleThumbCount(articles[i].Id.Hex())
		jsonStructs[i].Image = articles[i].Image
		jsonStructs[i].Reviews = redis.ArticleReviewCount(articles[i].Id.Hex())
	}

	respData := make(map[string]interface{})
	respData["page_number"] = form.PageNumber
	respData["page_more"] = DefaultPageSize*(form.PageNumber+1) < total
	respData["total"] = total
	respData["articles_without_content"] = jsonStructs
	writeResponse(request.RequestURI, resp, respData, err)
}

type articleInfoForm struct {
	Id          string `form:"article_id" binding:"required"`
	PubTime     bool   `form:"bl_publish_time"`
	Title       bool   `form:"bl_title"`
	Source      bool   `form:"bl_source"`
	ThumbCount  bool   `form:"bl_thumb_count"`
	Image       bool   `form:"bl_frist_image"`
	Content     bool   `form:"bl_content"`
	AccessToken string `form:"access_token"`
}

func articleInfoHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleInfoForm) {
	article := models.Article{}
	jsonStruct := &articleJsonStruct{}

	data := redis.GetArticle(form.Id)
	if len(data) > 0 && json.Unmarshal([]byte(data), jsonStruct) == nil {
		jsonStruct.Reviews = redis.ArticleReviewCount(form.Id)
		jsonStruct.Thumbs = redis.ArticleThumbCount(form.Id)

		writeResponse(request.RequestURI, resp, jsonStruct, errors.NoError)
		//log.Println("find article", form.Id, "in cache")
		return
	}

	if find, err := article.FindById(form.Id); !find {
		if err == errors.NoError {
			err = errors.NotExistsError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStruct.Id = article.Id.Hex()
	jsonStruct.Title = article.Title
	jsonStruct.Source = article.Source
	jsonStruct.Url = article.Url
	jsonStruct.PubTime = article.PubTime.Format(TimeFormat)
	jsonStruct.Reviews = redis.ArticleReviewCount(form.Id)
	jsonStruct.Thumbs = redis.ArticleThumbCount(form.Id)
	jsonStruct.Image = article.Image

	contents := make([]contentObject, len(article.Content))

	for i, text := range article.Content {
		if strings.HasPrefix(text, "[img]") &&
			strings.HasSuffix(text, "[img]") {
			fid := strings.TrimSuffix(strings.TrimPrefix(text, "[img]"), "[img]")
			contents[i] = contentObject{ContentType: "image",
				ContentText: imageUrl(fid, ImageThumbnail),
				ImageUrl:    imageUrl(fid, ImageOriginal),
			}
		} else if strings.HasPrefix(text, "http") &&
			strings.HasSuffix(text, ".jpg") {
			contents[i] = contentObject{ContentType: "image",
				ContentText: text,
				ImageUrl:    text,
			}
		} else {
			contents[i] = contentObject{ContentType: "text",
				ContentText: text,
			}
		}
	}
	jsonStruct.Content = contents
	writeResponse(request.RequestURI, resp, jsonStruct, errors.NoError)

	redis.LogArticleView(form.Id)

	if data, err := json.Marshal(jsonStruct); err == nil {
		redis.LogArticle(form.Id, string(data))
	} else {
		log.Println(err)
	}

}

type articleThumbForm struct {
	ArticleId   string      `form:"article_id" json:"article_id" binding:"required"`
	Status      bool        `form:"thumb_status" json:"thumb_status"`
	AccessToken string      `form:"access_token" json:"access_token" binding:"required"`
	User        models.User `form"-" json:"-"`
}

func (form *articleThumbForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ArticleId) {
		e.Fields["id"] = "invalid article id"
		return
	}
	form.User = userAuth(form.AccessToken, e)
}

func articleSetThumbHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleThumbForm) {
	var article models.Article
	article.Id = bson.ObjectIdHex(form.ArticleId)
	err := article.SetThumb(form.User.Userid, form.Status)

	writeResponse(request.RequestURI, resp, nil, err)

	if err == errors.NoError {
		redis.LogArticleThumb(form.ArticleId, form.Status)
	}
}

func checkArticleThumbHandler(request *http.Request, resp http.ResponseWriter, form articleThumbForm) {
	var article models.Article
	article.Id = bson.ObjectIdHex(form.ArticleId)
	thumbed, err := article.IsThumbed(form.User.Userid)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
	}

	respData := make(map[string]bool, 1)
	respData["is_thumbed"] = thumbed
	writeResponse(request.RequestURI, resp, respData, err)
}
