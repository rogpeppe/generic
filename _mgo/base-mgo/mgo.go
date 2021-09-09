package mgo

import "fmt"

func Dial(addr string) *Session {
	return &Session{addr}
}

type Session struct {
	addr string
}

func (s *Session) DB(name string) *Database {
	return &Database{
		session: s,
		name:    name,
	}
}

type Database struct {
	session *Session
	name    string
}

func (db *Database) C(name string) *Collection {
	return &Collection{
		db:   db,
		name: name,
	}
}

type Collection struct {
	db   *Database
	name string
}

func (c *Collection) Find(query interface{}) *Query {
	return &Query{
		collection: c,
		query:      query,
	}
}

type Query struct {
	collection *Collection
	query      interface{}
}

func (q *Query) Iter() *Iter {
	return &Iter{
		query: q,
	}
}

type Iter struct {
	query *Query
	item  int
}

func (it *Iter) Next(x interface{}) bool {
	sp, ok := x.(*string)
	if !ok {
		panic("unexpected type")
	}
	if it.item >= 5 {
		return false
	}
	*sp = fmt.Sprintf("%s.%s.%s query %#v; index %d",
		it.query.collection.db.session.addr,
		it.query.collection.db.name,
		it.query.collection.name,
		it.query.query,
		it.item,
	)
	it.item++
	return true
}
