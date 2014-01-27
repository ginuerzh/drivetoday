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
	//"github.com/ginuerzh/drivetoday/models"
	"github.com/nu7hatch/gouuid"
	"io"
	//"io/ioutil"
	"github.com/ginuerzh/weedo"
	//"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	TimeFormat      = "2006-01-02 15:04:05"
	DefaultPageSize = 10
)

type ImageSize int

const (
	ImageOriginal ImageSize = iota
	ImageBig
	ImageThumbnail
	ImageMedium
	ImageSmall
)

const (
	GuestUserPrefix = "guest:"

	UserTypeEmail = "usrpass"
	UserTypeWeibo = "weibo"
	UserTypeGuest = "guest"
)

const (
	AccessRate = 1 << iota // 001
	ThumbRate              // 010
	ReviewRate             // 100

	AccessRateMask = 6 // 110
	ThumbRateMask  = 5 // 101
	ReviewRateMask = 3 // 011
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

func writeResponse(uri string, resp http.ResponseWriter, data interface{}, errId int) []byte {
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	r, _ := jsonData(uri, data, errId)
	if errId == errs.DbError {
		resp.WriteHeader(http.StatusInternalServerError)
	}
	if errId == errs.NotFoundError {
		resp.WriteHeader(http.StatusNotFound)
	}
	if errId == errs.AccessError {
		resp.WriteHeader(http.StatusUnauthorized)
	}
	resp.Write(r)

	return r
}

func writeRawResponse(resp http.ResponseWriter, raw []byte) {
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.Write(raw)
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

/*
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
*/
func imageUrl(fid string, size ImageSize) string {
	var url string
	var err error
	s := strings.Split(fid, ",")
	if len(s) < 2 {
		return ""
	}

	id, _ := strconv.ParseUint(s[0], 10, 64)
	if url, _, err = weedo.Lookup(id); err != nil {
		return url
	}

	baseUrl := "http://" + url + "/" + s[0] + "/" + s[1]

	if size == ImageOriginal {
		if len(s) >= 3 {
			return baseUrl + "/" + s[2] + ".jpg"
		}
		return baseUrl + ".jpg"
	}

	if size == ImageBig {
		if len(s) >= 4 {
			return baseUrl + "_" + strconv.Itoa(int(size)) + "/" + s[3] + ".jpg"
		}
	}

	if size == ImageThumbnail {
		if len(s) >= 5 {
			return baseUrl + "_" + strconv.Itoa(int(size)) + "/" + s[4] + ".jpg"
		}
	}

	return baseUrl + "_" + strconv.Itoa(int(size)) + ".jpg"
}
