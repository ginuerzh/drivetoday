// article
package models

import (
	"github.com/ginuerzh/drivetoday/errors"
	//"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

type Article struct {
	Id         bson.ObjectId `bson:"_id,omitempty"`
	Title      string
	Source     string
	Tid        string
	Url        string
	Author     string
	AuthorPage string    `bson:"author_page"`
	PubTime    time.Time `bson:"pub_time"`
	Content    []string
	Thumbs     []string `bson:",omitempty"`
	Image      string
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

func (this *Article) ReviewCount() (int, int) {
	total := 0
	err := search(reviewColl, bson.M{"article_id": this.Id.Hex()}, nil, 0, 0, nil, &total, nil)
	if err != nil {
		return 0, errors.DbError
	}
	return total, errors.NoError
}

func GetArticleList(skip, limit int) (articles []Article, errId int) {
	err := search(articleColl, nil, nil, skip, limit, []string{"-pub_time"}, nil, &articles)
	if err != nil {
		return nil, errors.DbError
	}

	errId = errors.NoError
	return
}

func GetBriefArticles(skip, limit int) (total int, articles []Article, errId int) {
	err := search(articleColl, nil, bson.M{"content": false, "thumbs": false}, skip, limit, []string{"-pub_time"}, &total, &articles)
	if err != nil {
		return 0, nil, errors.DbError
	}

	errId = errors.NoError
	return
}
