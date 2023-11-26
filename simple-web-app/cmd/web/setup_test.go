package main

import (
	"log"
	"os"
	"simple-web-app/pkg/db"
	"testing"
)

var app application

/*
This function test main is it will always be executed before the actual tests run.
So goes tooling will actually look for the existence of a setup, underscore Tesco file and look for
the test main function and if it finds it, it runs that.
And then this function on line 13, it runs all of my tests.

When we need to do things like
implement sessions, configure database, all of that
configuration right here in the test main function.
*/
func TestMain(m *testing.M) {
	pathToTemplates = "./../../templates/"

	app.Session = getSession()

	//
	app.DSN = "host=localhost port=5432 user=postgres password=postgres dbname=users sslmode=disable timezone=UTC connect_timeout=5"
	conn, err := app.connectToDB()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	app.DB = db.PostgresConn{DB: conn}

	os.Exit(m.Run())
}
