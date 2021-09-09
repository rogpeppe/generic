package main

import "fmt"

func mgo2Dial(addr string) *mgo2Session {
	return &mgo2Session{addr}
}

type mgo2Session struct {
	addr string
}

func (s *mgo2Session) DB(name string) *mgo2Database {
	return &mgo2Database{
		session: s,
		name:    name,
	}
}

type mgo2Database struct {
	session *mgo2Session
	name    string
}

func (db *mgo2Database) C(name string) *mgo2Collection {
	return &mgo2Collection{
		db:   db,
		name: name,
	}
}

type mgo2Collection struct {
	db   *mgo2Database
	name string
}

func (c *mgo2Collection) Find(query interface{}) *mgo2Query {
	return &mgo2Query{
		collection: c,
		query:      query,
	}
}

type mgo2Query struct {
	collection *mgo2Collection
	query      interface{}
}

func (q *mgo2Query) Iter() *mgo2Iter {
	return &mgo2Iter{
		query: q,
	}
}

type mgo2Iter struct {
	query *mgo2Query
	item  int
}

func (it *mgo2Iter) Next(x interface{}) bool {
	sp, ok := x.(*string)
	if !ok {
		panic("unexpected type")
	}
	if it.item >= 5 {
		return false
	}
	*sp = fmt.Sprintf("%s.%s.%s query2 %#v; index %d",
		it.query.collection.db.session.addr,
		it.query.collection.db.name,
		it.query.collection.name,
		it.query.query,
		it.item,
	)
	it.item++
	return true
}
