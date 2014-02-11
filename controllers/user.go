// user
package controllers

import (
	"encoding/json"
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	UserRegisterV1Uri = "/1/account/register"
	UserLoginV1Uri    = "/1/account/login"
	UserLogoutV1Uri   = "/1/user/logout"
	UserInfoV1Uri     = "/1/user/getInfo"
	SetProfileV1Uri   = "/1/user/set_profile_image"
	UserNewsV1Uri     = "/1/user/news"
	UserListV1Uri     = "/1/users"

	UserArticlesV1Uri = "/1/user/article/:type/:id"
)

const (
	WeiboUserShowUrl  = "https://api.weibo.com/2/users/show.json"
	WeiboStatusUpdate = "https://api.weibo.com/2/statuses/update.json"
)

var (
	random *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func BindUserApi(m *martini.ClassicMartini) {
	m.Post(UserRegisterV1Uri, binding.Json(userRegForm{}), ErrorHandler, registerHandler)
	m.Post(UserLoginV1Uri, binding.Json(loginForm{}), ErrorHandler, loginHandler)
	m.Post(UserLogoutV1Uri, binding.Json(logoutForm{}), ErrorHandler, logoutHandler)
	m.Get(UserInfoV1Uri, binding.Form(getInfoForm{}), ErrorHandler, userInfoHandler)
	m.Post(SetProfileV1Uri, binding.Json(setProfileForm{}), ErrorHandler, setProfileHandler)
	//m.Get(UserNewsV1Uri, binding.Form(userNewsForm{}), ErrorHandler, userNewsHandler)
	m.Get(UserListV1Uri, binding.Form(userListForm{}), ErrorHandler, userListHandler)

	m.Get(UserArticlesV1Uri, userArticlesHandler)
}

// user register parameter
type userRegForm struct {
	Email    string `json:"email" binding:"required"`
	Nickname string `json:"nikename" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

func registerHandler(request *http.Request, resp http.ResponseWriter, redis *models.RedisLogger, form userRegForm) {
	var user models.User

	user.Userid = strings.ToLower(form.Email)
	user.Nickname = form.Nickname
	user.Password = Md5(form.Password)
	user.Role = form.Role
	user.RegTime = time.Now()
	//user.LastAccess = time.Now()
	//user.Online = true

	if exists, _ := user.CheckExists(); exists {
		writeResponse(request.RequestURI, resp, nil, errors.UserExistError)
		return
	}

	if err := user.Save(); err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
	} else {
		accessToken := Uuid()
		data := map[string]string{"access_token": accessToken}
		writeResponse(request.RequestURI, resp, data, err)

		redis.LogRegister(user.Userid)
		redis.LogOnlineUser(accessToken, user.Userid)
	}
}

// user login parameter
type loginForm struct {
	Userid   string `json:"userid"`
	Password string `json:"verfiycode"`
	Type     string `json:"account_type" binding:"required"`
}

type weiboInfo struct {
	ScreenName  string `json:"screen_name"`
	Gender      string `json:"gender"`
	Url         string `json:"url"`
	Avatar      string `json:"avatar_large"`
	Location    string `json:"location"`
	Description string `json:"description"`
	ErrorDesc   string `json:"error"`
	ErrCode     int    `json:"error_code"`
}

func weiboLogin(uid, password string, redis *models.RedisLogger) (*models.User, int) {
	weibo := weiboInfo{}
	user := &models.User{}

	v := url.Values{}
	v.Set("uid", uid)
	v.Set("access_token", password)

	url := WeiboUserShowUrl + "?" + v.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.HttpError
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.HttpError
	}

	if err := json.Unmarshal(data, &weibo); err != nil {
		return nil, errors.HttpError
	}

	if weibo.ErrCode != 0 {
		log.Println(weibo.ErrorDesc)
		return nil, errors.AccessError
	}

	user.Userid = strings.ToLower(uid)
	user.Password = Md5(password)
	exist, e := user.Exists()
	if e != errors.NoError {
		return nil, e
	}

	if exist {
		user.ChangePassword(user.Password)
		return user, errors.NoError
	}

	user.Nickname = weibo.ScreenName
	user.Gender = weibo.Gender
	user.Url = weibo.Url
	user.Profile = weibo.Avatar
	user.Location = weibo.Location
	user.About = weibo.Description
	user.Role = UserTypeWeibo
	user.RegTime = time.Now()

	if err := user.Save(); err != errors.NoError {
		return nil, err
	}
	redis.LogRegister(user.Userid)

	return user, errors.NoError
}
func guestLogin(redis *models.RedisLogger) (*models.User, int) {
	user := &models.User{}
	//user.Role = UserTypeGuest
	//user.RegTime = time.Now()
	user.Userid = models.GuestUserPrefix + strconv.Itoa(time.Now().Nanosecond()) + ":" + strconv.Itoa(random.Intn(65536))
	/*
		if err := user.Save(); err != errors.NoError {
			return nil, err
		}
		redis.LogRegister(user.Userid)
	*/

	return user, errors.NoError
}

func loginHandler(request *http.Request, resp http.ResponseWriter, form loginForm, redis *models.RedisLogger) {
	var user *models.User
	var err int
	accessToken := Uuid()

	if form.Type == UserTypeWeibo {
		user, err = weiboLogin(form.Userid, form.Password, redis)
	} else if form.Type == UserTypeGuest {
		user, err = guestLogin(redis)
		accessToken = models.GuestUserPrefix + accessToken // start with 'guest:' for redis checking
	} else if form.Type == UserTypeEmail {
		var find bool
		if find, err = user.FindByUserPass(strings.ToLower(form.Userid), Md5(form.Password)); !find {
			if err == errors.NoError {
				err = errors.AuthError
			}
		}
	}

	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	data := map[string]string{"access_token": accessToken}
	writeResponse(request.RequestURI, resp, data, errors.NoError)

	redis.LogOnlineUser(accessToken, user.Userid)
}

type logoutForm struct {
	AccessToken string `json:"access_token" binding:"required"`
}

func logoutHandler(request *http.Request, resp http.ResponseWriter, redis *models.RedisLogger, form logoutForm) {
	redis.DelOnlineUser(form.AccessToken)
	writeResponse(request.RequestURI, resp, nil, errors.NoError)

}

type getInfoForm struct {
	Userid string `form:"userid" json:"userid" binding:"required"`
}

type userJsonStruct struct {
	Userid   string `json:"userid"`
	Nickname string `json:"nikename"`
	Type     string `json:"account_type"`
	Phone    string `json:"phone_number"`
	About    string `json:"about"`
	Location string `json:"location"`
	Profile  string `json:"profile_image"`
	RegTime  string `json:"register_time"`
	Views    int64  `json:"view_count"`
	Thumbs   int64  `json:"thumb_count"`
	Reviews  int64  `json:"review_count"`
	Online   bool   `json:"online"`
}

func userInfoHandler(request *http.Request, resp http.ResponseWriter, form getInfoForm, redis *models.RedisLogger) {
	var user models.User

	if find, err := user.FindByUserId(form.Userid); !find {
		if err == errors.NoError {
			err = errors.NotFoundError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	respData := make(map[string]interface{})
	respData["userid"] = user.Userid
	respData["nikename"] = user.Nickname
	respData["account_type"] = user.Role
	respData["phone_number"] = user.Phone
	respData["about"] = user.About
	respData["location"] = user.Location
	respData["profile_image"] = user.Profile
	respData["register_time"] = user.RegTime.Format(TimeFormat)
	respData["online"] = redis.IsOnline(user.Userid)

	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type setProfileForm struct {
	ImageId     string `json:"image_id" binding:"required"`
	AccessToken string `json:"access_token"  binding:"required"`
	//User        models.User `json:"-"`
}

func setProfileHandler(request *http.Request, resp http.ResponseWriter, redis *models.RedisLogger, form setProfileForm) {
	var user models.User

	userid := redis.OnlineUser(form.AccessToken)
	if len(userid) == 0 {
		writeResponse(request.RequestURI, resp, nil, errors.AccessError)
		return
	}

	user.Userid = userid
	err := user.ChangeProfile(form.ImageId)
	writeResponse(request.RequestURI, resp, nil, err)
}

type userNewsForm struct {
	AccessToken string `form:"access_token" json:"access_token"  binding:"required"`
}

func userNewsHandler(request *http.Request, resp http.ResponseWriter, form userNewsForm) {
	writeResponse(request.RequestURI, resp, nil, errors.NoError)
}

type userListForm struct {
	PageNumber  int    `form:"page_number" json:"page_number"`
	AccessToken string `form:"access_token" json:"access_token"`
}

func userListHandler(request *http.Request, resp http.ResponseWriter, redis *models.RedisLogger, form userListForm) {
	pageSize := DefaultPageSize + 3
	total, users, err := models.UserList(pageSize*form.PageNumber, pageSize)
	if err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	jsonStructs := make([]userJsonStruct, len(users))
	for i, _ := range users {
		view, thumb, review, _ := users[i].ArticleCount()

		jsonStructs[i].Userid = users[i].Userid
		jsonStructs[i].Nickname = users[i].Nickname
		jsonStructs[i].Type = users[i].Role
		jsonStructs[i].Profile = users[i].Profile
		jsonStructs[i].Phone = users[i].Phone
		jsonStructs[i].Location = users[i].Location
		jsonStructs[i].About = users[i].About
		jsonStructs[i].RegTime = users[i].RegTime.Format(TimeFormat)
		jsonStructs[i].Views = view
		jsonStructs[i].Thumbs = thumb
		jsonStructs[i].Reviews = review
		jsonStructs[i].Online = redis.IsOnline(users[i].Userid)
	}

	respData := make(map[string]interface{})
	respData["page_number"] = form.PageNumber
	respData["page_more"] = pageSize*(form.PageNumber+1) < total
	respData["total"] = total
	respData["users"] = jsonStructs
	writeResponse(request.RequestURI, resp, respData, err)
}

func userArticlesHandler(request *http.Request, resp http.ResponseWriter, params martini.Params, redis *models.RedisLogger) {
	articleType := params["type"]
	userid := params["id"]

	user := models.User{Userid: userid}

	var ids []string

	switch articleType {
	case "view":
		ids, _ = user.RatedArticles(models.AccessRate)
	case "thumb":
		ids, _ = user.RatedArticles(models.ThumbRate)
	case "review":
		ids, _ = user.RatedArticles(models.ReviewRate)
	}

	articles, err := models.GetArticles(ids...)
	jsonStructs := make([]articleJsonStruct, len(articles))
	for i, _ := range articles {
		view, thumb, review := redis.ArticleCount(articles[i].Id.Hex())

		jsonStructs[i].Id = articles[i].Id.Hex()
		jsonStructs[i].Title = articles[i].Title
		jsonStructs[i].Source = articles[i].Source
		jsonStructs[i].Url = articles[i].Url
		jsonStructs[i].PubTime = articles[i].PubTime.Format(TimeFormat)
		jsonStructs[i].Image = imageUrl(articles[i].Image, ImageThumbnail)
		jsonStructs[i].Views = int(view)
		jsonStructs[i].Thumbs = int(thumb)
		jsonStructs[i].Reviews = int(review)
	}

	writeResponse(request.RequestURI, resp, jsonStructs, err)
}
