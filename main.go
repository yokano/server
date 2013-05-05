package okanoworld

import(
	"net/http"
	"strconv"
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"fmt"
	"log"
)

func init() {
	log.Println("init")
	
	// ランキング
	http.HandleFunc("/getranking", getRanking)
	http.HandleFunc("/putranking", putRanking)
	
	// 無茶振りBacklog
	http.HandleFunc("/backlog", requestBacklog);
}

/**
 * ランキングデータの型
 * @member {string} Name プレイヤー名
 * @member {int} Score 得点
 */
type Entity struct {
	Name string
	Score int
}

/**
 * ランキングを取得する
 * @function
 */
func getRanking(w http.ResponseWriter, r *http.Request) {
	var query *datastore.Query
	var kind string
	var entity *Entity
	var limit int
	var err error
	var c appengine.Context
	var iterator *datastore.Iterator
	var count int
	var i int
	var entities []*Entity
	var result []byte

	c = appengine.NewContext(r)
	
	kind = r.FormValue("kind")
	limit, err = strconv.Atoi(r.FormValue("limit"))
	check(c, err)
	
	query = datastore.NewQuery(kind).Limit(limit).Order("-Score")
	count, err = query.Count(c)
	check(c, err)
	
	iterator = query.Run(c)
	entities = make([]*Entity, 0)
	for i = 0; i < count; i++ {
		entity = new(Entity)
		_, err = iterator.Next(entity)
		check(c, err)
		entities = append(entities, entity)
	}
	
	result, err = json.Marshal(entities)
	check(c, err)
	
	fmt.Fprintf(w, "%s", result)
}

/**
 * ランキングに登録する
 * @function
 */
func putRanking(w http.ResponseWriter, r *http.Request) {
	var kind string
	var name string
	var score int
	var err error
	var c appengine.Context
	var key *datastore.Key
	var entity *Entity
	
	c = appengine.NewContext(r)
	
	kind = r.FormValue("kind")
	name = r.FormValue("name")
	
	score, err = strconv.Atoi(r.FormValue("score"))
	check(c, err)
	
	key = datastore.NewIncompleteKey(c, kind, nil)
	
	entity = new(Entity)
	entity.Name = name
	entity.Score = score
	
	_, err = datastore.Put(c, key, entity)
	check(c, err)
}

/**
 * エラーがあればコンソールに出力する
 * @function
 * @param {appengine.Context} c コンテキスト
 * @param {error} err チェックするエラーオブジェクト
 */
func check(c appengine.Context, err error) {
	if err != nil {
		c.Errorf(err.Error())
	}
}