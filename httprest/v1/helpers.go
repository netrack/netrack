package httprest

import (
	"net/http"

	"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
)

func Format(r *http.Request) (format.ReadFormatter, format.WriteFormatter) {
	return ReadFormat(r), WriteFormat(r)
}

func ReadFormat(r *http.Request) format.ReadFormatter {
	header := r.Header.Get(httputil.HeaderContentType)

	// If header is not provided, but this function called
	// (Content-Type filter pass this request), that means,
	// request was sent without body, so return nil formatter.
	if header == "" {
		return nil
	}

	f, err := format.Format(header)
	if err != nil {
		log.FatalLog("helpers/READ_FORMAT",
			"Failed to select read formatter for request: ", err)
	}

	return f
}

func WriteFormat(r *http.Request) format.WriteFormatter {
	f, err := format.Format(r.Header.Get(httputil.HeaderAccept))
	if err != nil {
		log.FatalLog("helpers/WRITE_FORMAT",
			"Failed to select write formatter for request: ", err)
	}

	return f
}
