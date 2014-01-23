// Event
package models

import (
	"encoding/json"
	"github.com/ginuerzh/drivetoday/errors"
	//"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

func init() {
	ensureIndex(eventColl, "owner", "-ctime")
}

type Event struct {
	Id        bson.ObjectId `bson:"_id,omitempty"`
	Type      string
	Ctime     time.Time
	ArticleId string `bson:"article_id"`
	User      string `bson:"user"`
	Owner     string
	Read      bool
	Message   string
}

type eventMessage struct {
	Type      string `json:"type"`
	Ctime     int64  `json:"ctime"`
	ArticleId string `json:"article_id"`
	User      string `json:"user"`
	Message   string `json:"message"`
}

func (em eventMessage) Json() string {
	data, err := json.Marshal(&em)
	if err != nil {
		log.Println(err)
	}
	return string(data)
}

func (this *Event) findOne(query interface{}) (bool, int) {
	var events []Event

	err := search(eventColl, query, nil, 0, 1, nil, nil, &events)
	if err != nil {
		return false, errors.DbError
	}
	if len(events) > 0 {
		*this = events[0]
	}

	return len(events) > 0, errors.NoError
}

func (this *Event) Save() (errId int) {
	errId = errors.NoError

	this.Id = bson.NewObjectId()
	if err := save(eventColl, this); err != nil {
		errId = errors.DbError
	}
	return
}

func (event Event) Json() string {
	eventMsg := eventMessage{}
	eventMsg.Type = event.Type
	eventMsg.Ctime = event.Ctime.Unix()
	eventMsg.ArticleId = event.ArticleId
	eventMsg.User = event.User
	eventMsg.Message = event.Message
	log.Println(eventMsg.Json())
	return eventMsg.Json()
}
