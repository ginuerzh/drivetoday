// article
package models

import (
	"github.com/ginuerzh/drivetoday/errors"
	//"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

func init() {
	ensureIndex(articleColl, "-pub_time")
}

type Article struct {
	Id         bson.ObjectId `bson:"_id,omitempty"`
	Title      string
	Source     string `bson:"from"`
	Tid        string
	Url        string
	Author     string
	AuthorPage string    `bson:"author_page"`
	PubTime    time.Time `bson:"pub_time"`
	Content    []string
	//Thumbs     []string `bson:",omitempty"`
	Image   string
	Random  int
	Publish bool
}

func (this *Article) findOne(query interface{}) (bool, int) {
	var articles []Article

	err := search(articleColl, query, nil, 0, 1, nil, nil, &articles)
	if err != nil {
		return false, errors.DbError
	}
	if len(articles) > 0 {
		*this = articles[0]
	}

	return len(articles) > 0, errors.NoError
}

/*
func (this *Article) SetThumb(userid string, thumb bool) int {
	var change bson.M

	if thumb {
		change = bson.M{
			"$addToSet": bson.M{
				"thumbs": userid,
			},
		}
	} else {
		change = bson.M{
			"$pull": bson.M{
				"thumbs": userid,
			},
		}
	}
	err := updateId(articleColl, this.Id, change)
	if err != nil {
		return errors.DbError
	}
	return errors.NoError
}

func (this *Article) IsThumbed(userid string) (bool, int) {
	count := 0
	err := search(articleColl, bson.M{"_id": this.Id, "thumbs": userid}, nil, 0, 0, nil, &count, nil)
	if err != nil {
		return false, errors.DbError
	}
	return count > 0, errors.NoError
}
*/

func (this *Article) FindById(id string) (bool, int) {
	return this.findOne(bson.M{"_id": bson.ObjectIdHex(id)})
}

func (this *Article) Reviews(skip, limit int) (total int, reviews []Review, errId int) {
	err := search(reviewColl, bson.M{"article_id": this.Id.Hex()}, nil, skip, limit, []string{"-ctime"}, &total, &reviews)
	if err != nil {
		return 0, nil, errors.DbError
	}

	errId = errors.NoError
	return
}

/*
func (this *Article) ReviewCount() (int, int) {
	total := 0
	err := search(reviewColl, bson.M{"article_id": this.Id.Hex()}, nil, 0, 0, nil, &total, nil)
	if err != nil {
		return 0, errors.DbError
	}
	return total, errors.NoError
}
*/
func (this *Article) LoadBrief() int {
	var articles []Article
	if err := search(articleColl, bson.M{"_id": this.Id}, bson.M{"content": false}, 0, 1, nil, nil, &articles); err != nil {
		return errors.DbError
	}

	if len(articles) > 0 {
		*this = articles[0]
	}
	return errors.NoError
}

func GetArticles(articleIds ...string) (articles []Article, errId int) {
	ids := make([]bson.ObjectId, len(articleIds))
	for i, _ := range articleIds {
		ids[i] = bson.ObjectIdHex(articleIds[i])
	}
	err := search(articleColl,
		bson.M{"_id": bson.M{"$in": ids}},
		bson.M{"content": false},
		0, 0, []string{"-pub_time"}, nil, &articles)
	if err != nil {
		return nil, errors.DbError
	}

	errId = errors.NoError
	return
}

func GetBriefArticles(skip, limit int) (total int, articles []Article, errId int) {
	err := search(articleColl, bson.M{"publish": true}, bson.M{"content": false}, skip, limit, []string{"-pub_time"}, &total, &articles)
	if err != nil {
		log.Println(err)
		return 0, nil, errors.DbError
	}

	errId = errors.NoError
	return
}

/*
func RandomArticles(excludes []string, max int) (article []Article, errId int) {
	ids := make([]bson.ObjectId, len(excludes))
	for i, _ := range excludes {
		ids[i] = bson.ObjectIdHex(excludes[i])
	}

	selector := bson.M{
		"_id":bson.M{"$nin": ids},
		"random":
	}
}
*/
