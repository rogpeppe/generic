package main

import (
	"fmt"
)

func main() {
	s := mgoDial("1.2.3.4")
	PrintBobDetails[*mgoSession, *mgoDatabase, *mgoCollection, *mgoQuery, *mgoIter](s)
}

type Mgo struct {
	type Session interface {
		DB(name) DB
	}
	type DB interface {
		C(name) Collection
	}
	type Collection interface {
		Query(q interface{}) Iter
	}
	type Iter interface {
		Next(into interface{}) bool
	}
}

type oldMgo = Mgo{
	Session: *mgo.Session,
}

func PrintBobDetails[m oldMgo](session m.Session) {

func PrintBobDetails[S Session[DB, C, Q, I], DB Database[C, Q, I], C Collection[Q, I], Q Query[I], I Iter](session S) {
	iter := session.DB("mydb").C("people").Find("some query").Iter()
	var s string
	for iter.Next(&s) {
		fmt.Println("got: ", s)
	}
}

type Session[DB Database[C, Q, I], C Collection[Q, I], Q Query[I], I Iter] interface {
	DB(name string) DB
}

type Database[C Collection[Q, I], Q Query[I], I Iter] interface{
	C(name string) C
}

type Collection[Q Query[I], I Iter] interface{
	Find(id interface{}) Q
}

type Query[I Iter] interface{
	Iter() I
}

type Iter interface {
	Next(result interface{}) bool
}

// GenericSession converts from a value of one specific session type to a generic version
// of it.
func GenericSession[S Session[DB, C, Q, I], DB Database[C, Q, I], C Collection[Q, I], Q Query[I], I Iter](s Session) genSession

type genSession interface {
	DB(name string) genDatabase
}

type genDatabase interface {
	C(name string) genCollection
}

type genCollection interface {
	Find(id interface{}) genQuery
}

type genQuery interface {
	Iter() Iter
}
