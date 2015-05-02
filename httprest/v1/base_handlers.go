package httprest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	// Register address management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewBaseHandler)
	mech.RegisterHTTPDriver(constructor)
}

type BaseHandler struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

func NewBaseHandler() mech.HTTPDriver {
	return &BaseHandler{}
}

func (h *BaseHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFilterFunc(h.acceptFilter)
	h.C.Mux.HandleFilterFunc(h.contentFilter)

	log.InfoLog("base_handlers/ENABLE_HOOK",
		"Base filters enabled")
}

func (h *BaseHandler) acceptFilter(rw http.ResponseWriter, r *http.Request) {
	f, err := format.Format(r.Header.Get(httputil.HeaderAccept))
	if err != nil {
		log.ErrorLog("base_handlers/ACCEPT_FILTER",
			"Failed to select Accept formatter for request: ", err)

		formats := strings.Join(format.FormatNameList(), ", ")
		body := models.Error{fmt.Sprintf("only '%s' are acceptable", formats)}

		f.Write(rw, body, http.StatusNotAcceptable)
	}
}

func (h *BaseHandler) contentFilter(rw http.ResponseWriter, r *http.Request) {
	if r.ContentLength == 0 {
		return
	}

	f, err := format.Format(r.Header.Get(httputil.HeaderContentType))
	if err != nil {
		log.ErrorLog("base_handlers/CONTENT_FILTER",
			"Failed to select ContentType formatter for request: ", err)

		formats := strings.Join(format.FormatNameList(), ", ")
		body := models.Error{fmt.Sprintf("only '%s' are supported", formats)}

		f.Write(rw, body, http.StatusUnsupportedMediaType)
	}
}
