package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/gorilla/context"
	"gopkg.in/mgo.v2"
)

func main() {

	// connect to the database
	db, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatal("cannot dial mongo", err)
	}
	defer db.Close() // clean up when we're done

	// Adapt our handle function using withDB
	h := Adapt(http.HandlerFunc(handle), withDB(db))

	// add the handler
	http.Handle("/comments", context.ClearHandler(h))

	// start the server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}

}

type Adapter func(http.Handler) http.Handler

func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

func handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleRead(w, r)
	case "POST":
		handleInsert(w, r)
	case "PUT":
		handleUpdate(w, r)
	case "DELETE":
		handleDelete(w, r)
	default:
		http.Error(w, "Not supported", http.StatusMethodNotAllowed)
	}
}

type comment struct {
	ID       bson.ObjectId `json:"id" bson:"_id"`
	Name     string        `json:"name" bson:"name"`
	Category string        `json:"category" bson:"category"`
	Price    float32       `json:"price" bson:"price"`
	Update   time.Time     `json:"update" bson:"update"`
	Quantity int           `json:"quantity" bson:"quantity"`
}

func handleInsert(w http.ResponseWriter, r *http.Request) {
	db := context.Get(r, "database").(*mgo.Session)

	// decode the request body
	var c comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// give the comment a unique ID
	c.ID = bson.NewObjectId()
	c.Update = time.Now()

	// insert it into the database
	if err := db.DB("commentsapp").C("comments").Insert(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// redirect to it
	http.Redirect(w, r, "/comments/"+c.ID.Hex(), http.StatusTemporaryRedirect)
}
func handleRead(w http.ResponseWriter, r *http.Request) {
	db := context.Get(r, "database").(*mgo.Session)

	// load the comments
	var comments []*comment
	if err := db.DB("commentsapp").C("comments").
		Find(nil).Sort("-when").Limit(100).All(&comments); err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// write it out
	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	db := context.Get(r, "database").(*mgo.Session)

	// decode the request body
	var c comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// give the comment a unique ID
	c.ID = bson.NewObjectId()
	c.Update = time.Now()

	var s comment
	if err := db.DB("commentsapp").C("comments").
		Find(c.ID).Sort("-when").Limit(1).All(&s); err != nil {

	}

	// insert it into the database
	if err := db.DB("commentsapp").C("comments").Update(&s, &c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// redirect to it
	http.Redirect(w, r, "/comments/"+c.ID.Hex(), http.StatusTemporaryRedirect)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	db := context.Get(r, "database").(*mgo.Session)

	// decode the request body
	var c comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// give the comment a unique ID
	c.ID = bson.NewObjectId()
	c.Update = time.Now()

	// insert it into the database
	if err := db.DB("commentsapp").C("comments").DropIndexName(c.ID.String()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// redirect to it
	http.Redirect(w, r, "/comments/"+c.ID.Hex(), http.StatusTemporaryRedirect)
}
func withDB(db *mgo.Session) Adapter {

	// return the Adapter
	return func(h http.Handler) http.Handler {

		// the adapter (when called) should return a new handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// copy the database session
			dbsession := db.Copy()
			defer dbsession.Close() // clean up

			// save it in the mux context
			context.Set(r, "database", dbsession)

			// pass execution to the original handler
			h.ServeHTTP(w, r)

		})
	}
}
