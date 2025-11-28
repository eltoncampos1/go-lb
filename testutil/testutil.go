package testutil

import (
	"net/http"
	"net/http/httptest"
)

func CreateTestServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

func CreateTestServerSimple() *httptest.Server {
	return CreateTestServer("", http.StatusOK)
}
