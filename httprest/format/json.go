package format

import (
	"encoding/json"
	"net/http"

	"github.com/netrack/netrack/httputil"
)

func init() {
	// Register formatter for application/json and */* types.
	Register(httputil.TypeApplicationJSON, &JSONFormatter{})
	Register(httputil.TypeAny, &JSONFormatter{})
}

// JSONFormatter formats data into JSON.
type JSONFormatter struct{}

// Read implements Formatter interface.
func (f *JSONFormatter) Read(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Add(httputil.HeaderAccept, httputil.TypeApplicationJSON)

	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// Write implements Formatter interface.
func (f *JSONFormatter) Write(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Add(httputil.HeaderContentType, httputil.TypeApplicationJSON)

	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
}
