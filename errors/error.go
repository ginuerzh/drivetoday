// error
package errors

import (
	"fmt"
)

const (
	NoError   = iota
	AuthError = 1000 + iota
	UserExistError
	AccessError
	DbError
	_
	JsonError
	UserNotFoundError
	PasswordError
	InvalidFileError
	HttpError
	FileNotFoundError
	_
	NotExistsError
	InvalidAddrError
	InvalidMsgError
	DeviceTokenError
	ReviewNotFoundError
	InviteCodeError
	FileTooLargeError
	FileUploadError
)

var errMap map[int]string = map[int]string{
	NoError:             "success",
	AuthError:           "auth error",
	UserExistError:      "user exists",
	AccessError:         "access token error",
	DbError:             "database error",
	JsonError:           "json data error",
	UserNotFoundError:   "user not found",
	PasswordError:       "password invalid",
	InvalidFileError:    "file invalid",
	HttpError:           "http error",
	FileNotFoundError:   "file not found",
	NotExistsError:      "not exists",
	InvalidAddrError:    "address invalid",
	InvalidMsgError:     "message invalid",
	DeviceTokenError:    "device token invalid",
	ReviewNotFoundError: "review not found",
	InviteCodeError:     "invite code invalid",
	FileTooLargeError:   "file too large",
	FileUploadError:     "file upload error",
}

func ErrString(id int) string {
	return errMap[id]
}

type Error struct {
	Id   int    `json:"error_id"`
	Desc string `json:"error_desc"`
}

func NewError(id int) Error {
	return Error{Id: id, Desc: ErrString(id)}
}

func (e Error) ErrString() string {
	return fmt.Sprintf("Err: %d, %s", e.Id, e.Desc)
}
