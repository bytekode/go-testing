package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"simple-web-app/pkg/data"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func Test_app_authenticate(t *testing.T) {
	var tests = []struct {
		name               string
		requestBody        string
		expectedStatusCode int
	}{
		{"valid user", `{"email":"admin@example.com","password":"secret"}`, http.StatusOK},
		{"not json", `I'm not json`, http.StatusUnauthorized},
		{"empty json", `{}`, http.StatusUnauthorized},
		{"empty email", `{"email":""}`, http.StatusUnauthorized},
		{"empty password", `{"email":"admin@example.com","password":""}`, http.StatusUnauthorized},
		{"invalid user", `{"email":"admin@someotherdomain.com","password":"secret"}`, http.StatusUnauthorized},
	}

	for _, test := range tests {
		var reader io.Reader
		reader = strings.NewReader(test.requestBody)
		req, _ := http.NewRequest("POST", "/auth", reader)
		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.authenticate)

		handler.ServeHTTP(rr, req)

		if test.expectedStatusCode != rr.Code {
			t.Errorf("%s: returned wrong status code; expected %d got %d", test.name, test.expectedStatusCode, rr.Code)
		}
	}
}

func Test_app_refresh(t *testing.T) {
	var tests = []struct {
		name               string
		token              string
		expectedStatusCode int
		resetRefreshTime   bool
	}{
		{"valid token", "", http.StatusOK, true},
		{"valid but not ready to expire", "", http.StatusTooEarly, false},
		{"expired token", expiredToken, http.StatusBadRequest, false},
	}

	testUser := data.User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "User",
		Email:     "admin@example.com",
	}

	oldRefreshTime := refreshTokenExpiry

	for _, test := range tests {
		var tkn string
		if test.token == "" {
			if test.resetRefreshTime {
				refreshTokenExpiry = time.Second * 1
			}
			tokens, _ := app.generateTokenPair(&testUser)
			tkn = tokens.RefreshToken
		} else {
			tkn = test.token
		}

		postedData := url.Values{
			"refresh_token": {tkn},
		}

		req, _ := http.NewRequest("POST", "/refresh-token", strings.NewReader(postedData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.refresh)
		handler.ServeHTTP(rr, req)

		if rr.Code != test.expectedStatusCode {
			t.Errorf("%s: expected status %d but got %d", test.name, test.expectedStatusCode, rr.Code)
		}
		refreshTokenExpiry = oldRefreshTime
	}
}

// when running unit test, we are not connecting to database
func Test_app_userHandlers(t *testing.T) {
	var tests = []struct {
		name           string
		method         string
		json           string
		paramID        string
		handler        http.HandlerFunc
		expectedStatus int
	}{
		{"allUsers", "GET", "", "", app.allUsers, http.StatusOK},
		{"deleteUser", "DELETE", "", "1", app.deleteUser, http.StatusNoContent},
		{"deleteUser bad url param", "DELETE", "", "y", app.deleteUser, http.StatusBadRequest},
		{"getUser valid", "GET", "", "1", app.getUser, http.StatusOK},
		{"getUser invalid", "GET", "", "100", app.getUser, http.StatusBadRequest},
		{"getUser bad url param", "GET", "", "y", app.getUser, http.StatusBadRequest},

		{
			"updateUser valid",
			"PATCH",
			`{"id":1,"first_name":"Administrator","last_name":"User","email":"admin@example.com"}`,
			"",
			app.updateUser,
			http.StatusNoContent,
		},
		{
			"updateUser invalid",
			"PATCH",
			`{"id":100,"first_name":"Administrator","last_name":"User","email":"admin@example.com"}`,
			"",
			app.updateUser,
			http.StatusBadRequest,
		},
		{
			"updateUser invalid json",
			"PATCH",
			`{"id":1,first_name:"Administrator","last_name":"User","email":"admin@example.com"}`,
			"",
			app.updateUser,
			http.StatusBadRequest,
		},
		{
			"insert valid",
			"PUT",
			`{"first_name":"Jack","last_name":"Smith","email":"jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusNoContent,
		},
		{
			"insert invalid",
			"PUT",
			`{"foo":"bar","first_name":"Jack","last_name":"Smith","email":"jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusBadRequest,
		},
		{
			"insert invalid json",
			"PUT",
			`{first_name:"Jack","last_name":"Smith","email":"jack@example.com"}`,
			"",
			app.insertUser,
			http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		var req *http.Request
		if test.json == "" {
			req, _ = http.NewRequest(test.method, "/", nil)
		} else {
			req, _ = http.NewRequest(test.method, "/", strings.NewReader(test.json))
		}
		if test.paramID != "" {
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("userID", test.paramID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(test.handler)
		handler.ServeHTTP(rr, req)

		if rr.Code != test.expectedStatus {
			t.Errorf("%s: wrong status retured; expected %d, got %d", test.name, test.expectedStatus, rr.Code)
		}
	}
}
