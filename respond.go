package respond

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"net/http"
)

var (
	mutex      sync.RWMutex
	responders = make(map[*http.Request]*Responder)
)

// Responder provides the ability to respond using repsond.With calls.
type Responder struct {
	// Encoder is a function field that gets the encoder to
	// use to respond to the specified http.Request.
	// By default, just returns the JSON Encoder.
	Encoder func(w http.ResponseWriter, r *http.Request) Encoder

	// OnErr is a function field that is called when an error occurs
	// during the responding process.
	//     - If Encoder.Encode returns an error
	// OnErrLog and OnErrPanic are two provided options with
	// OnErrLog being the default.
	OnErr func(w http.ResponseWriter, r *http.Request, err error)

	// Transform changes the status and data before it is written.
	// By default, has no effect.
	// Useful for handling different types of data differently (like errors)
	// or enveloping the response.
	Transform func(w http.ResponseWriter, r *http.Request, status int, data interface{}) (int, interface{})
}

// New makes a new Responder.
func New() *Responder {
	return &Responder{
		Encoder: func(http.ResponseWriter, *http.Request) Encoder { return JSON },
		OnErr:   OnErrLog,
	}
}

// With writes a response.
func With(w http.ResponseWriter, r *http.Request, status int, data interface{}) {

	// get the responder for this request
	mutex.RLock()
	responder, ok := responders[r]
	mutex.RUnlock()
	if !ok {
		panic("respond: must wrap with Handler or HandlerFunc")
	}

	// optionally transform the data
	if responder.Transform != nil {
		status, data = responder.Transform(w, r, status, data)
	}

	// write the response
	w.WriteHeader(status)
	encoder := responder.Encoder(w, r)
	w.Header().Set("Content-Type", encoder.ContentType(w, r))
	if err := encoder.Encode(w, r, data); err != nil {
		responder.OnErr(w, r, err)
	}
}

// OnErrLog logs the error using log.Println.
func OnErrLog(_ http.ResponseWriter, _ *http.Request, err error) {
	log.Println("respond:", err)
}

// OnErrPanic panics with the specified error.
func OnErrPanic(_ http.ResponseWriter, _ *http.Request, err error) {
	panic(fmt.Sprint("respond:", err))
}

// Handler wraps an http.Handler and enables its handlers to use
// respond.With.
func (d *Responder) Handler(handler http.Handler) http.Handler {
	return d.HandlerFunc(handler.ServeHTTP)
}

// HandlerFunc wraps an http.HandlerFunc and enables the handler to
// use repsond.With.
func (d *Responder) HandlerFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		responders[r] = d
		mutex.Unlock()
		defer func() {
			mutex.Lock()
			delete(responders, r)
			mutex.Unlock()
		}()
		fn(w, r)
	}
}

// Encoder descirbes an object capable of encoding
// a response.
type Encoder interface {
	// Encode writes a serialization of v to w, optionally using additional
	// information from the http.Request to do so.
	Encode(w http.ResponseWriter, r *http.Request, v interface{}) error
	// ContentType gets a string that will become the Content-Type header
	// when responding through w to the specified http.Request.
	// Most of the time the argument will be ignored, but occasionally
	// details in the request, or even in the headers in the ResponseWriter may
	// change the content type.
	ContentType(w http.ResponseWriter, r *http.Request) string
}

type jsonEncoder struct{}

var _ Encoder = (*jsonEncoder)(nil)

// JSON is an Encoder for JSON.
var JSON *jsonEncoder

func (*jsonEncoder) Encode(w http.ResponseWriter, r *http.Request, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func (*jsonEncoder) ContentType(w http.ResponseWriter, r *http.Request) string {
	return "application/json; charset=utf-8"
}
