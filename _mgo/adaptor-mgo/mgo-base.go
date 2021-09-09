package main

import "fmt"

func mgoDial(addr string) *mgoSession {
	return &mgoSession{addr}
}

type mgoSession struct {
	addr string
}

func (s *mgoSession) DB(name string) *mgoDatabase {
	return &mgoDatabase{
		session: s,
		name:    name,
	}
}

type mgoDatabase struct {
	session *mgoSession
	name    string
}

func (db *mgoDatabase) C(name string) *mgoCollection {
	return &mgoCollection{
		db:   db,
		name: name,
	}
}

type mgoCollection struct {
	db   *mgoDatabase
	name string
}

func (c *mgoCollection) Find(query interface{}) *mgoQuery {
	return &mgoQuery{
		collection: c,
		query:      query,
	}
}

type mgoQuery struct {
	collection *mgoCollection
	query      interface{}
}

func (q *mgoQuery) Iter() *mgoIter {
	return &mgoIter{
		query: q,
	}
}

type mgoIter struct {
	query *mgoQuery
	item  int
}

func (it *mgoIter) Next(x interface{}) bool {
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
