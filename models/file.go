// file
package models

import (
	"github.com/ginuerzh/drivetoday/errors"
	"github.com/ginuerzh/weedo"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

func init() {
	ensureIndex(fileColl, "fid")
	ensureIndex(fileColl, "-uploadDate")
}

type File struct {
	Id          bson.ObjectId `bson:"_id,omitempty"`
	Fid         string
	Name        string `bson:"filename"`
	Length      int64  `bson:"length"`
	Md5         string
	Owner       string
	Count       int
	ContentType string    `bson:"contentType"`
	UploadDate  time.Time `bson:"uploadDate"`
}

func (this *File) Exists() (bool, int) {
	count := 0
	err := search(fileColl, bson.M{"fid": this.Fid}, nil, 0, 0, nil, &count, nil)
	if err != nil {
		return false, errors.DbError
	}
	return count > 0, errors.NoError
}

func (this *File) findOne(query interface{}) (bool, int) {
	var files []File

	err := search(fileColl, query, nil, 0, 1, nil, nil, &files)
	if err != nil {
		return false, errors.DbError
	}
	if len(files) > 0 {
		*this = files[0]
	}

	return len(files) > 0, errors.NoError
}

func (this *File) FindByFid(fid string) (bool, int) {
	return this.findOne(bson.M{"fid": fid})
}

func (this *File) Save() (errId int) {
	errId = errors.NoError

	this.Id = bson.NewObjectId()
	if err := save(fileColl, this); err != nil {
		errId = errors.DbError
	}
	return
}

func (this *File) Delete() (errId int) {
	errId = errors.NoError

	remove := func(c *mgo.Collection) error {
		err := c.Remove(bson.M{"fid": this.Fid})
		if err == nil {
			weedo.Delete(this.Fid, this.Count) //TODO: fail process
		}
		return err
	}

	if err := withCollection(fileColl, nil, remove); err != nil {
		if err != mgo.ErrNotFound {
			errId = errors.DbError
		}
	}
	return
}

func (this *File) OwnedBy(userid string) (bool, int) {
	return this.findOne(bson.M{"fid": this.Fid, "owner": userid})
}
