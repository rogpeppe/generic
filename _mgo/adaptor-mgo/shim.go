package fakemgo

import "gopkg.in/mgo.v2"

func NewSession(s *mgo.Session) Session {
	return sessionShim{s}
}

type sessionShim struct {
	*mgo.Session
}

func (s sessionShim) DB(name string) Database {
	return databaseShim{s.Session.DB(name)}
}

type databaseShim struct {
	*mgo.Database
}

func (db databaseShim) C(name string) Collection {
	return collectionShim{db.Database.C(name)}
}

type collectionShim struct {
	*mgo.Collection
}

func (c collectionShim) Find(query interface{}) Query {
	return queryShim{c.Collection.Find(query)}
}

type queryShim struct {
	*mgo.Query
}

func (q queryShim) Iter() Iter {
	return q.Query.Iter()
}
