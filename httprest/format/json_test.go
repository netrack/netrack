package format

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netrack/netrack/httputil"
)

func TestJSONFormatter(t *testing.T) {
	f := JSONFormatter{}

	r, _ := http.NewRequest("PUT", "http://127.0.0.1/resource", nil)
	rw := httptest.NewRecorder()

	err := f.Write(rw, r, map[string]string{"status": "alive"})
	if err != nil {
		t.Fatal("Failed to write data in JSON format:", err)
	}

	header := rw.Header().Get(httputil.HeaderContentType)
	if header != httputil.TypeApplicationJSON {
		t.Fatal("Expected Content-Type header in a response")
	}

	if rw.Body.Len() == 0 {
		t.Fatal("Invalid data in body")
	}
}
