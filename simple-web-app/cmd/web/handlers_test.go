package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
)

func Test_application_handlers(t *testing.T) {
	var theTests = []struct {
		name                    string
		url                     string
		expectedStatusCode      int
		expectedURL             string
		expectedFirstStatusCode int
	}{
		{"home", "/", http.StatusOK, "/", http.StatusOK},
		{"404", "/fish", http.StatusNotFound, "/fish", http.StatusNotFound},
		{"profile", "/user/profile", http.StatusOK, "/", http.StatusTemporaryRedirect},
	}

	routes := app.routes()

	// create a test server
	ts := httptest.NewTLSServer(routes)
	defer ts.Close()

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, e := range theTests {
		resp, err := ts.Client().Get(ts.URL + e.url)
		if err != nil {
			t.Log(err)
			t.Fatal(err)
		}

		if resp.StatusCode != e.expectedStatusCode {
			t.Errorf("For %s: expected{%d}, but got {%d}", e.name, e.expectedStatusCode, resp.StatusCode)
		}

		if resp.Request.URL.Path != e.expectedURL {
			t.Errorf("%s: expected final url of %s but got %s", e.name, e.expectedURL, resp.Request.URL.Path)
		}

		resp2, _ := client.Get(ts.URL + e.url)
		if resp2.StatusCode != e.expectedFirstStatusCode {
			t.Errorf("%s: expected first returned status code to be %d but got %d", e.name, e.expectedFirstStatusCode, resp2.StatusCode)
		}
	}
}

func TestAppHome(t *testing.T) {
	var tests = []struct {
		name         string
		putInSession string
		expectedHTML string
	}{
		{"first visit", "", "<small>From Session:"},
		{"second visit", "hello, world!", "<small>From Session: hello, world!"},
	}

	for _, test := range tests {
		req, _ := http.NewRequest("GET", "/", nil)
		req = addContextAndSessionToReq(req, app)
		_ = app.Session.Destroy(req.Context())

		if test.putInSession != "" {
			app.Session.Put(req.Context(), "test", test.putInSession)
		}

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.Home)

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("TestAppHome retured wrong status code; exptected 200 but got %d", rr.Code)
		}

		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), test.expectedHTML) {
			t.Errorf("%s: did not find %s in response body", test.name, test.expectedHTML)
		}
	}
}

func TestApp_renderWithBadTemplate(t *testing.T) {
	// set template path to a location with a bad template
	pathToTemplates = "./testdata/"

	req, _ := http.NewRequest("GET", "/", nil)
	req = addContextAndSessionToReq(req, app)
	rr := httptest.NewRecorder()

	err := app.render(rr, req, "bad.page.gohtml", &TemplateData{})
	if err == nil {
		t.Error("Expected error from bad template, but did not get one")
	}
	pathToTemplates = "./../../templates/"
}

func getCtx(req *http.Request) context.Context {
	ctx := context.WithValue(req.Context(), contextUserKey, "unknown")
	return ctx
}

func addContextAndSessionToReq(req *http.Request, app application) *http.Request {
	req = req.WithContext(getCtx(req))
	ctx, _ := app.Session.Load(req.Context(), req.Header.Get("X-Session"))
	return req.WithContext(ctx)
}

func Test_app_Login(t *testing.T) {
	var tests = []struct {
		name               string
		postedData         url.Values
		expectedStatusCode int
		expectedLoc        string
	}{
		{
			name:               "valid login",
			postedData:         url.Values{"email": {"admin@example.com"}, "password": {"secret"}},
			expectedStatusCode: http.StatusSeeOther, expectedLoc: "/user/profile",
		},
		{
			name:               "missing form data",
			postedData:         url.Values{"email": {""}, "password": {""}},
			expectedStatusCode: http.StatusSeeOther, expectedLoc: "/",
		},
		{
			name:               "user not found",
			postedData:         url.Values{"email": {"you@there.com"}, "password": {"secret"}},
			expectedStatusCode: http.StatusSeeOther, expectedLoc: "/",
		},
		{
			name:               "bad credentials",
			postedData:         url.Values{"email": {"admin@example.com"}, "password": {"password"}},
			expectedStatusCode: http.StatusSeeOther, expectedLoc: "/",
		},
		{
			name:               "user not found",
			postedData:         url.Values{"email": {"admin2@example.com"}, "password": {"secret"}},
			expectedStatusCode: http.StatusSeeOther, expectedLoc: "/",
		},
	}

	for _, test := range tests {
		req, _ := http.NewRequest("POST", "/login", strings.NewReader(test.postedData.Encode()))
		req = addContextAndSessionToReq(req, app)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.Login)
		handler.ServeHTTP(rr, req)

		if rr.Code != test.expectedStatusCode {
			t.Errorf("%s: returned wrong status codel; epected %d, but got %d", test.name, test.expectedStatusCode, rr.Code)
		}

		actualLoc, err := rr.Result().Location()
		if err == nil {
			if actualLoc.String() != test.expectedLoc {
				t.Errorf("%s: epected location %s, but got %s", test.name, test.expectedLoc, actualLoc.String())
			}
		} else {
			t.Errorf("%s: no location header set", test.name)
		}
	}
}

func Test_app_UploadFiles(t *testing.T) {
	// set up pipes
	pr, pw := io.Pipe()

	// create a new writer, of type *io.Writer
	writer := multipart.NewWriter(pw)

	// create a waitgroup and add 1 to it
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// simulate uploading a file using go routine and our writer
	go simulatePNGUpload("./testdata/uploads/img.png", writer, t, wg)

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// call app.UploadFiles
	uploadedFiles, err := app.UploadFiles(request, "./testdata/uploads")
	if err != nil {
		t.Error(err)
	}

	// perform our tests
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName))
}

func simulatePNGUpload(fileToUpload string, writer *multipart.Writer, t *testing.T, wg *sync.WaitGroup) {
	defer writer.Close()
	defer wg.Done()

	// create fome data field `file` with value being filename
	part, err := writer.CreateFormFile("file", path.Base(fileToUpload))
	if err != nil {
		t.Error(err)
	}

	// open the actual file
	f, err := os.Open(fileToUpload)
	if err != nil {
		log.Fatalf(err.Error())
		t.Error(err)
	}
	defer f.Close()

	// decode the image
	img, _, err := image.Decode(f)
	if err != nil {
		t.Error("error decoding image:", err)
	}

	// write PNG to io.Writer
	if part == nil {
		log.Fatalf("part is nil")
		return
	}

	if img == nil {
		log.Fatalf("img is nil")
		return
	}
	err = png.Encode(part, img)
	if err != nil {
		t.Error(err)
	}
}
