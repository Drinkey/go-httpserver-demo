package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpServerResponseHeader(t *testing.T) {
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
	if string(body) != "jjok" {
		t.Errorf("body error: %s", string(body))
	}
	if result.Header.Get(headerKey) != headerValue {
		t.Errorf("error when valdating response header, %v", result.Header)
	}
}
