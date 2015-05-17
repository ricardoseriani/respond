package respond_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cheekybits/is"
	"github.com/matryer/respond"
)

var testdata = map[string]interface{}{"test": true}

func newTestRequest() *http.Request {
	r, err := http.NewRequest("GET", "Something", nil)
	if err != nil {
		panic("bad request: " + err.Error())
	}
	return r
}

type testHandler struct{}

func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	respond.With(w, r, http.StatusOK, testdata)
}

func TestRespondWithHandlerFunc(t *testing.T) {
	is := is.New(t)

	fn := func(w http.ResponseWriter, r *http.Request) {
		respond.With(w, r, http.StatusOK, testdata)
	}

	responder := respond.New()
	w := httptest.NewRecorder()
	r := newTestRequest()

	responder.HandlerFunc(fn)(w, r)

	// assert it was written
	is.Equal(http.StatusOK, w.Code)
	var data map[string]interface{}
	is.NoErr(json.Unmarshal(w.Body.Bytes(), &data))
	is.Equal(data, testdata)
	is.Equal(w.HeaderMap.Get("Content-Type"), "application/json; charset=utf-8")
}

func TestToWithHandler(t *testing.T) {
	is := is.New(t)

	responder := respond.New()
	w := httptest.NewRecorder()
	r := newTestRequest()

	handler := &testHandler{}
	responder.Handler(handler).ServeHTTP(w, r)

	is.Equal(http.StatusOK, w.Code)
	var data map[string]interface{}
	is.NoErr(json.Unmarshal(w.Body.Bytes(), &data))
	is.Equal(data, testdata)
}

func TestTransform(t *testing.T) {
	is := is.New(t)

	newData := map[string]interface{}{"changed": true}

	responder := respond.New()
	responder.Transform = func(w http.ResponseWriter, r *http.Request, status int, data interface{}) (int, interface{}) {
		return http.StatusCreated, newData
	}
	w := httptest.NewRecorder()
	r := newTestRequest()

	handler := &testHandler{}
	responder.Handler(handler).ServeHTTP(w, r)

	is.Equal(http.StatusCreated, w.Code)
	var data map[string]interface{}
	is.NoErr(json.Unmarshal(w.Body.Bytes(), &data))
	is.Equal(data, newData)
}

func TestJSON(t *testing.T) {
	is := is.New(t)

	w := httptest.NewRecorder()
	r := newTestRequest()

	is.Equal(respond.JSON.ContentType(w, r), "application/json; charset=utf-8")

}
