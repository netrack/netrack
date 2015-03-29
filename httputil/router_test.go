package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMux(t *testing.T) {
	mux := NewServeMux()

	tenantsIndex := func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("tenant index"))
	}

	tenantsShow := func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("tenant show"))
	}

	usersIndex := func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("user index"))
	}

	usersShow := func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("user show"))
	}

	notFound := func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("not found"))
	}

	mux.HandleFunc("GET", "/tenants", tenantsIndex)
	mux.HandleFunc("GET", "/tenants/{tenant}", tenantsShow)
	mux.HandleFunc("GET", "/tenants/{tenant}/users", usersIndex)
	mux.HandleFunc("GET", "/tenants/{tenant}/users/{user}", usersShow)
	mux.NotFound = http.HandlerFunc(notFound)

	test := func(url, body string) {
		r, _ := http.NewRequest("GET", url, nil)
		rw := httptest.NewRecorder()

		mux.ServeHTTP(rw, r)

		if rw.Body.String() != body {
			t.Fatal("Failed to send request to right handler:", url, rw.Body.String())
		}
	}

	test("/tenants", "tenant index")
	test("/tenants/123", "tenant show")
	test("/tenants/123/users", "user index")
	test("/tenants/123/users/456", "user show")
	test("/tenants/123/users/456/logout", "not found")
	test("/customers/123", "not found")
}
