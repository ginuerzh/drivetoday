// article
package controllers

import (
	//"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"labix.org/v2/mgo/bson"
	"log"
	//slopeone "github.com/ginuerzh/go-slope-one"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	ArticleListV1Uri     = "/1/article/timelines"
	ArticleInfoV1Uri     = "/1/article/get"
	ArticleSetThumbV1Uri = "/1/article/thumb"
	ArticleThumbedV1Uri  = "/1/article/is_thumbed"
	ArticleRelatedV1Uri  = "/1/article/related_articles"

	SlopeOneUrl = "http://localhost:8090/slopeone"
)

func BindArticleApi(m *martini.ClassicMartini) {
	m.Get(ArticleListV1Uri, binding.Form(articleListForm{}), ErrorHandler, articleListHandler)
	m.Get(ArticleInfoV1Uri, binding.Form(articleInfoForm{}), ErrorHandler, articleInfoHandler)

	m.Post(ArticleSetThumbV1Uri, binding.Json(articleThumbForm{}), ErrorHandler, articleSetThumbHandler)
	m.Get(ArticleThumbedV1Uri, binding.Form(articleThumbForm{}), ErrorHandler, checkArticleThumbHandler)
	m.Get(ArticleRelatedV1Uri, binding.Form(relatedArticleForm{}), ErrorHandler, relatedArticleHandler)
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
	Read    bool            `json:"read_status"`
	Content []contentObject `json:"content"`
}

func articleSource(s string) string {
	if s == "autohome" {
		return "汽车之家"
	} else if s == "bitauto" {
		return "易车网"
	} else {
		return s
	}
}

func articleListHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleListForm) {
	total, articles, err := models.GetBriefArticles(DefaultPageSize*form.PageNumber, DefaultPageSize)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	var reads []bool
	ids := make([]string, len(articles))
	for i, _ := range articles {
		ids[i] = articles[i].Id.Hex()
	}

	userid := redis.OnlineUser(form.AccessToken)
	if reads = redis.ArticleView(userid, ids...); reads == nil {
		reads = make([]bool, len(articles))
	}

	jsonStructs := make([]articleJsonStruct, len(articles))
	for i, _ := range articles {
		jsonStructs[i].Id = ids[i]
		jsonStructs[i].Title = articles[i].Title
		jsonStructs[i].Source = articleSource(articles[i].Source)
		jsonStructs[i].Url = articles[i].Url
		jsonStructs[i].PubTime = articles[i].PubTime.Format(TimeFormat)
		jsonStructs[i].Thumbs = redis.ArticleThumbCount(articles[i].Id.Hex())
		jsonStructs[i].Image = imageUrl(articles[i].Image, ImageThumbnail)
		jsonStructs[i].Reviews = redis.ArticleReviewCount(articles[i].Id.Hex())
		jsonStructs[i].Read = reads[i]
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
	AccessToken string `form:"access_token" binding:"required"`
}

func articleInfoHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleInfoForm) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	user := models.User{Userid: userid}
	user.RateArticle(form.Id, AccessRate, false)
	redis.LogArticleView(form.Id, userid)

	article := models.Article{}
	jsonStruct := &articleJsonStruct{}

	data := redis.GetArticleCache(form.Id)
	if len(data) > 0 {
		writeRawResponse(resp, data)
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
	jsonStruct.Source = articleSource(article.Source)
	jsonStruct.Url = article.Url
	jsonStruct.PubTime = article.PubTime.Format(TimeFormat)
	jsonStruct.Reviews = redis.ArticleReviewCount(form.Id)
	jsonStruct.Thumbs = redis.ArticleThumbCount(form.Id)

	contents := make([]contentObject, len(article.Content))

	for i, text := range article.Content {

		if strings.HasPrefix(text, "[img]") && strings.HasSuffix(text, "[img]") {
			continue
		}

		if strings.HasPrefix(text, "[fid]") &&
			strings.HasSuffix(text, "[fid]") {
			fid := strings.TrimSuffix(strings.TrimPrefix(text, "[fid]"), "[fid]")
			contents[i] = contentObject{ContentType: "image",
				ContentText: imageUrl(fid, ImageThumbnail),
				ImageUrl:    imageUrl(fid, ImageOriginal),
			}
		} else {
			contents[i] = contentObject{ContentType: "text",
				ContentText: text,
			}
		}
	}
	jsonStruct.Content = contents
	raw := writeResponse(request.RequestURI, resp, jsonStruct, errors.NoError)
	redis.LogArticleCache(form.Id, raw)
}

type articleThumbForm struct {
	ArticleId   string `form:"article_id" json:"article_id" binding:"required"`
	Status      bool   `form:"thumb_status" json:"thumb_status"`
	AccessToken string `form:"access_token" json:"access_token" binding:"required"`
}

func (form *articleThumbForm) Validate(e *binding.Errors, req *http.Request) {
	if !bson.IsObjectIdHex(form.ArticleId) {
		e.Fields["id"] = "invalid article id"
		return
	}
}

func articleSetThumbHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form articleThumbForm) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	//var article models.Article
	//article.Id = bson.ObjectIdHex(form.ArticleId)
	//err := article.SetThumb(userid, form.Status)

	user := models.User{Userid: userid}
	if form.Status {
		user.RateArticle(form.ArticleId, ThumbRate, false)
	} else {
		user.RateArticle(form.ArticleId, ThumbRateMask, true)
	}

	redis.LogArticleThumb(userid, form.ArticleId, form.Status)

	writeResponse(request.RequestURI, resp, nil, errors.NoError)
}

func checkArticleThumbHandler(request *http.Request, resp http.ResponseWriter, form articleThumbForm, redis *RedisLogger) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}
	/*
		var article models.Article
		article.Id = bson.ObjectIdHex(form.ArticleId)
		thumbed, err := article.IsThumbed(userid)
		if err != errors.NoError {
			writeResponse(request.RequestURI, resp, nil, err)
		}
	*/

	respData := map[string]bool{"is_thumbed": redis.ArticleThumbed(userid, form.ArticleId)}
	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type relatedArticleForm struct {
	ArticleId   string `form:"article_id" json:"article_id"`
	AccessToken string `form:"access_token" json:"access_token" binding:"required"`
}

/*
func relatedArticleHandler(request *http.Request, resp http.ResponseWriter, form relatedArticleForm, redis *RedisLogger) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}
	articleIds := redis.RelatedArticles(form.ArticleId, 3)

	articles, err := models.GetArticles(articleIds...)

	jsonStructs := make([]articleJsonStruct, len(articles))

	for i, _ := range articles {
		jsonStructs[i].Id = articles[i].Id.Hex()
		jsonStructs[i].Title = articles[i].Title
		jsonStructs[i].Source = articles[i].Source
		jsonStructs[i].Url = articles[i].Url
		jsonStructs[i].PubTime = articles[i].PubTime.Format(TimeFormat)
		jsonStructs[i].Image = imageUrl(articles[i].Image, ImageThumbnail)
		//jsonStructs[i].Thumbs = redis.ArticleThumbCount(articles[i].Id.Hex())
		//jsonStructs[i].Reviews = redis.ArticleReviewCount(articles[i].Id.Hex())
		//jsonStructs[i].Read = reads[i]
	}

	respData := make(map[string]interface{})
	respData["related_articles"] = jsonStructs
	writeResponse(request.RequestURI, resp, respData, err)
}
*/
func relatedArticleHandler(request *http.Request, resp http.ResponseWriter, form relatedArticleForm, redis *RedisLogger) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}
	mRate := make(map[string]int)

	user := models.User{Userid: userid}
	if userRate, err := user.ArticleRate(); err == errors.NoError {
		for _, rate := range userRate.Rates {
			mRate[rate.Article] = rate.Rate
		}
	}
	//log.Println(mRate)
	data, err := json.Marshal(&mRate)
	if err != nil {
		log.Println(err)
	}
	r, err := http.Post(SlopeOneUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
		writeResponse(request.RequestURI, resp, nil, errors.DbError)
		return
	}
	defer r.Body.Close()

	data, err = ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		writeResponse(request.RequestURI, resp, nil, errors.DbError)
		return
	}

	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		log.Println(err)
	}
	//log.Println(ids)
	if len(ids) > 3 {
		ids = ids[:3]
	}
	articles, e := models.GetArticles(ids...)

	jsonStructs := make([]articleJsonStruct, len(articles))
	for i, _ := range articles {
		jsonStructs[i].Id = articles[i].Id.Hex()
		jsonStructs[i].Title = articles[i].Title
		//jsonStructs[i].Source = articles[i].Source
		//jsonStructs[i].Url = articles[i].Url
		//jsonStructs[i].PubTime = articles[i].PubTime.Format(TimeFormat)
		//jsonStructs[i].Image = imageUrl(articles[i].Image, ImageThumbnail)
		//jsonStructs[i].Thumbs = redis.ArticleThumbCount(articles[i].Id.Hex())
		//jsonStructs[i].Reviews = redis.ArticleReviewCount(articles[i].Id.Hex())
		//jsonStructs[i].Read = reads[i]
	}

	respData := make(map[string]interface{})
	respData["related_articles"] = jsonStructs
	writeResponse(request.RequestURI, resp, respData, e)
}
