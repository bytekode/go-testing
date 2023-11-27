package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"simple-web-app/pkg/data"
	"testing"
)

func Test_app_enableCORS(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	var tests = []struct {
		name         string
		method       string
		expectHeader bool
	}{
		{"preflight", "OPTIONS", true},
		{"get", "GET", false},
	}

	for _, test := range tests {
		handlerToTest := app.enableCORS(nextHandler)
		req := httptest.NewRequest(test.method, "http://testing", nil)
		rr := httptest.NewRecorder()
		handlerToTest.ServeHTTP(rr, req)

		if test.expectHeader && rr.Header().Get("Access-Control-Allow-Credentials") == "" {
			t.Errorf("%s: expected header; but did not got one", test.name)
		}

		if !test.expectHeader && rr.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Errorf("%s: did not expected header; but got one", test.name)
		}
	}

}

func Test_app_authRequired(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	testUser := data.User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "Admin",
		Email:     "admin@example.com",
	}

	tokens, _ := app.generateTokenPair(&testUser)

	var tests = []struct {
		name             string
		token            string
		expectAuthorized bool
		setHeader        bool
	}{
		{"valid token", fmt.Sprintf("Bearer %s", tokens.Token), true, true},
		{"expired token", fmt.Sprintf("Bearer %s", expiredToken), false, true},
		{"no token", "", false, false},
	}

	for _, test := range tests {
		req, _ := http.NewRequest("GET", "/", nil)
		if test.setHeader {
			req.Header.Set("Authorization", test.token)
		}

		rr := httptest.NewRecorder()
		handlerToTest := app.authRequired(nextHandler)
		handlerToTest.ServeHTTP(rr, req)

		if test.expectAuthorized && rr.Code == http.StatusUnauthorized {
			t.Errorf("%s: got code 401, and should not have", test.name)
		}

		if !test.expectAuthorized && rr.Code != http.StatusUnauthorized {
			t.Errorf("%s: did not get code 401, and should not have", test.name)
		}
	}
}
