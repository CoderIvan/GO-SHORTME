package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
)

const (
	expTime       = 60
	longURL       = "https://www.baidu.com"
	shortLink     = "IFHzaO"
	shortLinkInfo = `{"url":"https://www.baidu.com","created_at":"2021-01-27 21:31:42.7385233 +0800 CST m=+13.468085701","expiration_in_minutes":60}`
)

type storageMock struct {
	mock.Mock
}

var app App
var mockR *storageMock

func (s *storageMock) Shorten(url string, exp int64) (string, error) {
	args := s.Called(url, exp)
	return args.String(0), args.Error(1)
}

func (s *storageMock) Unshorten(eid string) (string, error) {
	args := s.Called(eid)
	return args.String(0), args.Error(1)
}

func (s *storageMock) ShortlinkInfo(eid string) (string, error) {
	args := s.Called(eid)
	return args.String(0), args.Error(1)
}

func init() {
	app = App{}
	mockR = new(storageMock)
	app.Initialize(&Env{S: mockR})
}

func TestCreateShortlink(t *testing.T) {
	var jsonStr = []byte(`{
		"url":"https://www.baidu.com",
		"expiration_in_minutes":60
	}`)

	req, err := http.NewRequest("POST", "/api/shorten", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal("Shoule be able to create a request.", err)
	}
	req.Header.Set("Content-Type", "application/json")

	mockR.On("Shorten", longURL, int64(expTime)).Return(shortLink, nil).Once()

	rw := httptest.NewRecorder()
	app.Router.ServeHTTP(rw, req)

	if rw.Code != http.StatusCreated {
		t.Fatalf("Excepted receive %d. Got %d", http.StatusCreated, rw.Code)
	}

	resp := struct {
		Shortlink string `json:"shortlink"`
	}{}

	if err := json.NewDecoder(rw.Body).Decode(&resp); err != nil {
		t.Fatalf("Should decode the response")
	}

	if resp.Shortlink != shortLink {
		t.Fatalf("Excepted receive %d. Got %d", http.StatusCreated, rw.Code)
	}
}

func TestRedirect(t *testing.T) {
	r := fmt.Sprintf("/api/%s", shortLink)
	req, err := http.NewRequest("GET", r, nil)
	if err != nil {
		t.Fatal("Should be able to create a request.", err)
	}

	mockR.On("Unshorten", shortLink).Return(longURL, nil).Once()
	rw := httptest.NewRecorder()
	app.Router.ServeHTTP(rw, req)

	if rw.Code != http.StatusTemporaryRedirect {
		t.Fatalf("Excepted receive %d. Got %d", http.StatusTemporaryRedirect, rw.Code)
	}
}
