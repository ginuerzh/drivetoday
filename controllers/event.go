// event
package controllers

import (
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"net/http"
)

const (
	EventListV1Uri     = "/1/event/timeline"
	NewEventCountV1Uri = "/1/event/news"
	EventReadV1Uri     = "/1/event/change_status_read"
)

func BindEventApi(m *martini.ClassicMartini) {
	m.Get(EventListV1Uri, binding.Form(eventListForm{}), ErrorHandler, eventListHandler)
	m.Get(NewEventCountV1Uri, binding.Form(eventListForm{}), ErrorHandler, newEventsHandler)
	m.Post(EventReadV1Uri, binding.Json(eventReadForm{}), ErrorHandler, eventReadHandler)
}

type eventListForm struct {
	PageNumber  int    `form:"page_number" json:"page_number"`
	AccessToken string `form:"access_token" json:"access_token" binding:"required"`
	//user        models.User `form:"-" json:"-"`
}

func (form *eventListForm) Validate(e *binding.Errors, req *http.Request) {
	//form.user = userAuth(form.AccessToken, e)
}

type eventJsonStruct struct {
	EventId   string `json:"event_id"`
	Type      string `json:"event_type"`
	Ctime     string `json:"publish_time"`
	ArticleId string `json:"article_id"`
	User      string `json:"publish_author"`
	Read      bool   `json:"read_status"`
	Message   string `json:"message"`
}

func eventListHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form eventListForm) {
	var user models.User

	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	user.Userid = userid
	total, events, err := user.Events(DefaultPageSize*form.PageNumber, DefaultPageSize)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStructs := make([]eventJsonStruct, len(events))
	for i, _ := range events {
		jsonStructs[i].EventId = events[i].Id.Hex()
		jsonStructs[i].Type = events[i].Type
		jsonStructs[i].Ctime = events[i].Ctime.Format(TimeFormat)
		jsonStructs[i].ArticleId = events[i].ArticleId
		jsonStructs[i].User = events[i].User
		jsonStructs[i].Read = true
		jsonStructs[i].Message = events[i].Message
	}

	respData := make(map[string]interface{})
	respData["page_number"] = form.PageNumber
	respData["page_more"] = DefaultPageSize*(form.PageNumber+1) < total
	respData["total"] = total
	respData["events"] = jsonStructs
	writeResponse(request.RequestURI, resp, respData, err)
}

func newEventsHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form eventListForm) {
	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	respData := map[string]interface{}{"events_count": redis.MessageCount(userid)}
	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type eventReadForm struct {
	AccessToken string   `form:"access_token" json:"access_token" binding:"required"`
	Ids         []string `form:"event_ids" json:"event_ids" binding:"required"`
}

func eventReadHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form eventReadForm) {
	//var user models.User

	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	redis.ClearMessages(userid)
	//user.Userid = userid
	//count, err := user.ReadEvents(form.Ids)

	//respData := map[string]interface{}{"read_count": count}
	writeResponse(request.RequestURI, resp, nil, errors.NoError)
}
