package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHttpServerResponseHeaderHealthzHandler(t *testing.T) {
	headerKey := "x-api-key"
	headerValue := "test-value-random-set"
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.Header.Add(headerKey, headerValue)

	w := httptest.NewRecorder()
	healthzHandler(w, r)
	result := w.Result()
	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Errorf("error occurs: %v\nbody=%s", err, string(body))
	}
	fmt.Println(string(body))
	if string(body) != "ok" {
		t.Errorf("body validation error: expect ok, but got %s", string(body))
	}
	if result.Header.Get(headerKey) != headerValue {
		t.Errorf("error when valdating response header, %v", result.Header)
	}
	if result.Header.Get("VERSION") != os.Getenv("VERSION") {
		t.Errorf("VERSION in header is %s, expect %s",
			result.Header.Get("VERSION"), os.Getenv("VERSION"))
	}
}

func TestHttpServerResponseHeaderDefaultHandler(t *testing.T) {
	headerKey := "x-api-key"
	headerValue := "test-value-random-set"
	r := httptest.NewRequest(http.MethodGet, "/heal", nil)
	r.Header.Add(headerKey, headerValue)

	w := httptest.NewRecorder()
	defaultHandler(w, r)
	result := w.Result()
	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Errorf("error occurs: %v\nbody=%s", err, string(body))
	}
	fmt.Println(string(body))
	if string(body) != "welcome" {
		t.Errorf("body validation error: expect welcome, but got %s", string(body))
	}
	if result.Header.Get(headerKey) != headerValue {
		t.Errorf("error when valdating response header, %v", result.Header)
	}
	if result.Header.Get("VERSION") != os.Getenv("VERSION") {
		t.Errorf("VERSION in header is %s, expect %s",
			result.Header.Get("VERSION"), os.Getenv("VERSION"))
	}
}
