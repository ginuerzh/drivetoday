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
}

// user register parameter
type userRegForm struct {
	Email    string `json:"email" binding:"required"`
	Nickname string `json:"nikename" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

func registerHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form userRegForm) {
	var user models.User

	user.Userid = strings.ToLower(form.Email)
	user.Nickname = form.Nickname
	user.Password = Md5(form.Password)
	user.Role = form.Role
	user.RegTime = time.Now()
	user.LastAccess = time.Now()
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

func weiboLogin(uid, password string, redis *RedisLogger) (*models.User, int) {
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

	user.Nickname = "weibo_" + weibo.ScreenName
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
func guestLogin(redis *RedisLogger) (*models.User, int) {
	user := &models.User{}
	//user.Role = UserTypeGuest
	//user.RegTime = time.Now()
	user.Userid = GuestUserPrefix + strconv.Itoa(time.Now().Nanosecond()) + ":" + strconv.Itoa(random.Intn(65536))
	/*
		if err := user.Save(); err != errors.NoError {
			return nil, err
		}
		redis.LogRegister(user.Userid)
	*/

	return user, errors.NoError
}

func loginHandler(request *http.Request, resp http.ResponseWriter, form loginForm, redis *RedisLogger) {
	var user *models.User
	var err int
	accessToken := Uuid()

	if form.Type == UserTypeWeibo {
		user, err = weiboLogin(form.Userid, form.Password, redis)
	} else if form.Type == UserTypeGuest {
		user, err = guestLogin(redis)
		accessToken = GuestUserPrefix + accessToken // start with 'guest:' for redis checking
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

func logoutHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form logoutForm) {
	redis.DelOnlineUser(form.AccessToken)
	writeResponse(request.RequestURI, resp, nil, errors.NoError)

}

type getInfoForm struct {
	Userid string `form:"userid" json:"userid" binding:"required"`
}

func userInfoHandler(request *http.Request, resp http.ResponseWriter, form getInfoForm) {
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

	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type setProfileForm struct {
	ImageId     string `json:"image_id" binding:"required"`
	AccessToken string `json:"access_token"  binding:"required"`
	//User        models.User `json:"-"`
}

func setProfileHandler(request *http.Request, resp http.ResponseWriter, redis *RedisLogger, form setProfileForm) {
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
