// user
package models

import (
	"github.com/ginuerzh/drivetoday/errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

type User struct {
	Id            bson.ObjectId `bson:"_id,omitempty"`
	Userid        string
	Password      string
	Nickname      string
	Gender        string    `bson:",omitempty"`
	Url           string    `bson:",omitempty"`
	Phone         string    `bson:",omitempty"`
	About         string    `bson:",omitempty"`
	Location      string    `bson:",omitempty"`
	Profile       string    `bson:",omitempty"`
	RegTime       time.Time `bson:"reg_time"`
	Role          string    `bson:",omitempty"`
	Online        bool
	LastAccess    time.Time `bson:"last_access"`
	ThumbArticles []string  `bson:"thumb_articles,omitempty"`
	AccessToken   string    `bson:"access_token,omitempty"`
}

func (this *User) Exists() (bool, int) {
	count := 0
	err := search(userCollection, bson.M{"userid": this.Userid}, nil, 0, 0, nil, &count, nil)
	if err != nil {
		return false, errors.DbError
	}
	return count > 0, errors.NoError
}

func (this *User) findOne(query interface{}) (bool, int) {
	var users []User

	err := search(userCollection, query, nil, 0, 1, nil, nil, &users)
	if err != nil {
		return false, errors.DbError
	}
	if len(users) > 0 {
		*this = users[0]
	}
	return len(users) > 0, errors.NoError
}

func (this *User) Save() (errId int) {
	errId = errors.NoError

	if exist, err := this.Exists(); exist {
		errId = err
		if errId == errors.NoError {
			errId = errors.UserExistError
		}
		return
	}

	insert := func(c *mgo.Collection) error {
		this.Id = bson.NewObjectId()
		return c.Insert(this)
	}

	if err := withCollection(userCollection, insert); err != nil {
		errId = errors.DbError
	}
	return
}

func (this *User) ChangePassword(newPass string) int {
	change := bson.M{
		"$set": bson.M{
			"password": newPass,
		},
	}

	err := updateId(userCollection, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *User) ChangeProfile(profile string) int {
	change := bson.M{
		"$set": bson.M{
			"profile": profile,
		},
	}

	err := updateId(userCollection, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *User) Upsert() (errId int) {
	errId = errors.NoError

	upsert := func(c *mgo.Collection) error {
		if len(this.Id.Hex()) == 0 {
			this.Id = bson.NewObjectId()
			this.RegTime = bson.Now()
		}
		_, err := c.UpsertId(this.Id, this)
		return err
	}

	if err := withCollection(userCollection, upsert); err != nil {
		errId = errors.DbError
	}

	return
}

func (this *User) Access() int {
	change := bson.M{
		"$set": bson.M{
			"last_access": time.Now(),
		},
	}
	err := updateId(userCollection, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *User) UpdateStatus() int {
	change := bson.M{
		"$set": bson.M{
			"online":       this.Online,
			"access_token": this.AccessToken,
			"last_access":  this.LastAccess,
		},
	}
	err := updateId(userCollection, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *User) FindByUserId(userid string) (bool, int) {
	return this.findOne(bson.M{"userid": userid})
}

func (this *User) FindByNickname(nickname string) (bool, int) {
	return this.findOne(bson.M{"nickname": nickname})
}

func (this *User) FindByUserPass(userid, password string) (bool, int) {
	return this.findOne(bson.M{"userid": userid, "password": password})
}

func (this *User) CheckExists() (bool, int) {
	type S []bson.M
	return this.findOne(bson.M{"$or": S{{"userid": this.Userid}, {"nickname": this.Nickname}}})
}

func (this *User) FindByAccessToken(accessToken string) (bool, int) {
	d, _ := time.ParseDuration("-30h") // auto logout after 30 minutes since last access

	return this.findOne(
		bson.M{
			"access_token": accessToken,
			"online":       true,
			"last_access": bson.M{
				"$gte": time.Now().Add(d),
			},
		})
}

func (this *User) Logout() int {
	this.Online = false
	this.LastAccess = bson.Now()
	this.AccessToken = ""

	change := bson.M{
		"$set": bson.M{
			"online":       this.Online,
			"access_token": "",
			"last_access":  this.LastAccess,
		},
	}

	err := updateId(userCollection, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *User) Reviews(skip, limit int) (total int, reviews []Review, errId int) {
	err := search(reviewColl, bson.M{"userid": this.Userid}, nil, skip, limit, []string{"-ctime"}, &total, &reviews)
	if err != nil {
		return 0, nil, errors.DbError
	}

	errId = errors.NoError
	return
}

func (this *User) Events(skip, limit int) (total int, events []Event, errId int) {
	err := search(eventColl, bson.M{"owner": this.Userid}, nil, skip, limit, []string{"-ctime"}, &total, &events)
	if err != nil {
		return 0, nil, errors.DbError
	}

	errId = errors.NoError
	return
}

func (this *User) NewEventCount() (count int, errId int) {
	err := search(eventColl, bson.M{"owner": this.Userid, "read": false}, nil, 0, 0, nil, &count, nil)
	if err != nil {
		return 0, errors.DbError
	}

	errId = errors.NoError
	return
}

func (this *User) ReadEvents(ids []string) (count int, errId int) {
	errId = errors.NoError

	selector := bson.M{
		"event_id": bson.M{
			"$in": ids,
		},
	}

	change := bson.M{
		"$set": bson.M{
			"read": true,
		},
	}

	update := func(c *mgo.Collection) error {
		info, err := c.UpdateAll(selector, change)
		count = info.Updated
		return err
	}

	if err := withCollection(eventColl, update); err != nil {
		errId = errors.DbError
	}

	return
}
