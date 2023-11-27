package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"simple-web-app/pkg/data"
	"testing"
)

func Test_app_getTokenFromHeaderAndVerify(t *testing.T) {
	testUser := data.User{
		ID:        1,
		FirstName: "Admin",
		LastName:  "Admin",
		Email:     "admin@example.com",
	}

	tokens, _ := app.generateTokenPair(&testUser)

	var tests = []struct {
		name          string
		token         string
		errorExpected bool
		setHeader     bool
		issuer        string
	}{
		{"valid", fmt.Sprintf("Bearer %s", tokens.Token), false, true, app.Domain},
		{"valid expired", fmt.Sprintf("Bearer %s", expiredToken), true, true, app.Domain},
		{"no header", "", true, false, app.Domain},
		{"invalid token", fmt.Sprintf("Bearer %s123", tokens.Token), true, true, app.Domain},
		{"no Bearer", fmt.Sprintf("Bear %s", tokens.Token), true, true, app.Domain},
		{"three header parts", fmt.Sprintf("Bearer %s 123", tokens.Token), true, true, app.Domain},

		// make sure it's the last one to run
		{"wrong issuer", fmt.Sprintf("Bearer %s", tokens.Token), true, true, "anotherdomain.com"},
	}

	for _, test := range tests {

		if test.issuer != app.Domain {
			app.Domain = test.issuer
			tokens, _ = app.generateTokenPair(&testUser)
		}

		req, _ := http.NewRequest("GET", "/", nil)
		if test.setHeader {
			req.Header.Add("Authorization", test.token)
		}

		rr := httptest.NewRecorder()
		_, _, err := app.getTokenFromHeaderAndVerify(rr, req)
		if err != nil && !test.errorExpected {
			t.Errorf("%s: did not expected error, but got one - %s", test.name, err.Error())
		}

		if err == nil && test.errorExpected {
			t.Errorf("%s: expected error, but did not get one - %s", test.name, err.Error())
		}

		app.Domain = "example.com"
	}
}
