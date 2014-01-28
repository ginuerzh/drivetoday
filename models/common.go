// common
package models

import (
	//"github.com/ginuerzh/drivetoday/errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

const (
	AccessRate = 1 << iota // 001
	ThumbRate              // 010
	ReviewRate             // 100

	AccessRateMask = 6 // 110
	ThumbRateMask  = 5 // 101
	ReviewRateMask = 3 // 011
)

var (
	mgoSession   *mgo.Session
	databaseName = "drivetoday"
	//databaseName   = "cartour"
	//articleColl    = "threads"
	userCollection = "users"
	articleColl    = "articles"
	reviewColl     = "reviews"
	fileColl       = "files"
	eventColl      = "events"
	rateColl       = "rates"
)

var (
	GuestUserPrefix = "guest:"
)

func getSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial("localhost")
		if err != nil {
			log.Println(err) // no, not really
		}
	}
	return mgoSession.Clone()
}

func withCollection(collection string, safe *mgo.Safe, s func(*mgo.Collection) error) error {
	session := getSession()
	defer session.Close()

	session.SetSafe(safe)
	c := session.DB(databaseName).C(collection)
	return s(c)
}

func search(collection string, query interface{}, selector interface{},
	skip, limit int, sortFields []string, total *int, result interface{}) error {

	q := func(c *mgo.Collection) error {
		qy := c.Find(query)
		var err error

		if selector != nil {
			qy = qy.Select(selector)
		}

		if total != nil {
			if *total, err = qy.Count(); err != nil {
				return err
			}
		}

		if limit > 0 {
			qy = qy.Limit(limit)
		}
		if skip > 0 {
			qy = qy.Skip(skip)
		}
		if len(sortFields) > 0 {
			qy = qy.Sort(sortFields...)
		}

		if result != nil {
			err = qy.All(result)
		}
		return err
	}

	return withCollection(collection, nil, q)
}

func updateId(collection string, id bson.ObjectId, change interface{}) error {
	update := func(c *mgo.Collection) error {
		return c.UpdateId(id, change)
	}

	return withCollection(collection, nil, update)
}

func update(collection string, selector, change interface{}, safe bool) error {
	update := func(c *mgo.Collection) error {
		return c.Update(selector, change)
	}
	if safe {
		return withCollection(collection, &mgo.Safe{}, update)
	}
	return withCollection(collection, nil, update)
}

func upsert(collection string, selector, change interface{}, safe bool) error {
	upsert := func(c *mgo.Collection) error {
		_, err := c.Upsert(selector, change)
		//log.Println(info, err)
		return err
	}
	if safe {
		return withCollection(collection, &mgo.Safe{}, upsert)
	}
	return withCollection(collection, nil, upsert)
}

func save(collection string, o interface{}) error {
	insert := func(c *mgo.Collection) error {
		return c.Insert(o)
	}

	return withCollection(collection, nil, insert)
}

func ensureIndex(collection string, keys ...string) error {
	ensure := func(c *mgo.Collection) error {
		return c.EnsureIndexKey(keys...)
	}

	return withCollection(collection, nil, ensure)
}

func DateString(t time.Time) string {
	return t.Format("2006-01-02")
}
