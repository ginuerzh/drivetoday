// file
package controllers

import (
	"github.com/codegangsta/martini"
	"github.com/codegangsta/martini-contrib/binding"
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/drivetoday/models"
	"github.com/ginuerzh/weedo"
	//"io"
	"time"
	//"labix.org/v2/mgo/bson"
	"log"
	"net/http"
)

const (
	FileUploadV1Uri    = "/1/file/upload"
	ImageDownloadV1Uri = "/1/image/get"
	FileDeleteV1Uri    = "/1/file/del"
)

func BindFileApi(m *martini.ClassicMartini) {
	m.Post(FileUploadV1Uri, binding.Form(fileUploadForm{}), ErrorHandler, fileUploadHandler)
	m.Get(ImageDownloadV1Uri, binding.Form(imageDownloadForm{}), ErrorHandler, imageDownloadHandler)
	m.Post(FileDeleteV1Uri, binding.Json(fileDeleteForm{}), ErrorHandler, fileDeleteHandler)
}

type fileUploadForm struct {
	AccessToken string      `form:"access_token" binding:"required"`
	user        models.User `form:"-"`
}

func (form *fileUploadForm) Validate(e *binding.Errors, req *http.Request) {
	log.Println(form.AccessToken)
	form.user = userAuth(form.AccessToken, e)
}

func fileUploadHandler(request *http.Request, resp http.ResponseWriter, form fileUploadForm) {
	filedata, header, err := request.FormFile("filedata")
	if err != nil {
		log.Println(err)
		writeResponse(request.RequestURI, resp, nil, errors.FileNotFoundError)
		return
	}

	fid, size, err := weedo.AssignUpload(header.Filename, header.Header.Get("Content-Type"), filedata)
	if err != nil {
		writeResponse(request.RequestURI, resp, nil, errors.FileUploadError)
		return
	}
	log.Println(fid, size, header.Filename, header.Header.Get("Content-Type"))

	filedata.Seek(0, 0)

	var file models.File
	file.Fid = fid
	file.Name = header.Filename
	file.ContentType = header.Header.Get("Content-Type")
	file.Size = size
	file.Md5 = FileMd5(filedata)
	file.Owner = form.user.Userid
	file.UploadDate = time.Now()
	if err := file.Save(); err != errors.NoError {
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	//url, _ := weedo.GetUrl(fid)
	respData := map[string]interface{}{"fileid": fid, "fileurl": imageUrl(fid, 0)}

	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type imageDownloadForm struct {
	ImageId   string `form:"image_id" binding:"required"`
	ImageSize string `form:"image_size_type"`
}

/*
func imageDownloadHandler(request *http.Request, resp http.ResponseWriter, form imageDownloadForm) {
	var file models.File

	if exist, err := file.FindByFid(form.ImageId); !exist {
		if err == errors.NoError {
			err = errors.FileNotFoundError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	fileData, err := weedo.Download(form.ImageId)
	if err != nil {
		writeResponse(request.RequestURI, resp, nil, errors.FileNotFoundError)
		return
	}
	defer fileData.Close()

	resp.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	io.Copy(resp, fileData)
}
*/

func imageDownloadHandler(request *http.Request, resp http.ResponseWriter, form imageDownloadForm) {
	url := imageUrl(form.ImageId, ImageOriginal)

	respData := map[string]string{"image_url": url}
	writeResponse(request.RequestURI, resp, respData, errors.NoError)
}

type fileDeleteForm struct {
	Fid         string      `json:"image_id" binding:"required"`
	AccessToken string      `json:"access_token" binding:"required"`
	user        models.User `json:"-"`
}

func (form *fileDeleteForm) Validate(e *binding.Errors, req *http.Request) {
	form.user = userAuth(form.AccessToken, e)
}

func fileDeleteHandler(request *http.Request, resp http.ResponseWriter, form fileDeleteForm) {
	var file models.File

	if find, err := file.FindByFid(form.Fid); !find {
		if err == errors.NoError {
			err = errors.FileNotFoundError
		}
		writeResponse(request.RequestURI, resp, nil, err)
		return
	}

	if file.Owner != form.user.Userid {
		writeResponse(request.RequestURI, resp, nil, errors.FileNotFoundError)
		return
	}

	err := file.Delete()
	writeResponse(request.RequestURI, resp, nil, err)
}
