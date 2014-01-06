// common
package controllers

import (
	"crypto/md5"
	"encoding/json"
	//"errors"
	"fmt"
	//simplejson "github.com/bitly/go-simplejson"
	"github.com/codegangsta/martini-contrib/binding"
	errs "github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"github.com/nu7hatch/gouuid"
	"io"
	//"io/ioutil"
	//"log"
	"github.com/ginuerzh/weedo"
	"net/http"
	"strconv"
)

const (
	TimeFormat      = "2006-01-02 15:04:05"
	DefaultPageSize = 10
)

type response struct {
	ReqPath  string      `json:"req_path"`
	RespData interface{} `json:"response_data"`
	Err      errs.Error  `json:"error"`
}

func jsonData(reqPath string, data interface{}, err int) ([]byte, error) {
	resp := response{ReqPath: reqPath, RespData: data, Err: errs.NewError(err)}
	return json.Marshal(resp)
}

func writeResponse(uri string, resp http.ResponseWriter, data interface{}, errId int) {
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	r, _ := jsonData(uri, data, errId)
	if errId == errs.DbError {
		resp.WriteHeader(http.StatusInternalServerError)
	}
	if errId == errs.FileNotFoundError {
		resp.WriteHeader(http.StatusNotFound)
	}
	resp.Write(r)
}

func Md5(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func FileMd5(file io.Reader) string {
	h := md5.New()
	io.Copy(h, file)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Uuid() string {
	u4, err := uuid.NewV4()
	if err != nil {
		fmt.Println("error:", err)
		return ""
	}

	return u4.String()
}

func ErrorHandler(err binding.Errors, request *http.Request, resp http.ResponseWriter) {
	if err.Count() > 0 {
		errId := errs.JsonError
		if _, ok := err.Fields["db"]; ok {
			errId = errs.DbError
		} else if _, ok = err.Fields["access"]; ok {
			errId = errs.AccessError
		}
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
		resp.WriteHeader(http.StatusBadRequest)
		r, _ := jsonData(request.RequestURI, nil, errId)
		resp.Write(r)
	}
}

func userAuth(accessToken string, e *binding.Errors) (user models.User) {
	find, err := user.FindByAccessToken(accessToken)
	if find {
		user.Access()
	}

	if !find || len(accessToken) == 0 {
		e.Fields["access"] = ""
	}
	if err == errs.DbError {
		e.Fields["db"] = ""
	}

	return
}

func imageUrl(fid string) string {
	var url string
	id, key, cookie, err := weedo.ParseFid(fid)
	if err != nil {
		return url
	}
	if url, err = weedo.Lookup(id); err != nil {
		return url
	}

	return "http://" + url + "/" + strconv.FormatUint(id, 10) + "/" +
		strconv.FormatUint(key, 16) + strconv.FormatUint(cookie, 16) + ".jpg"
}
