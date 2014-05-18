package httpspy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Table is a simple middleware http.Handler. It attempts to serve the request
// with a sequence of http.Handler types. If no handlers respond a 404 (not
// found) response is returned.
type Table []http.Handler

func (t Table) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	spy := NewSpy(resp)
	for i := range t {
		t[i].ServeHTTP(spy, req)
		if spy.Code() != 0 {
			return
		}
	}
	http.NotFound(resp, req)
}

// MyService is a simple HTTP service. It has two routes
//	POST /puppy
//	POST /kitty
func MyService() http.Handler {
	var idcount int64
	type Pet struct {
		Id   int64  `json:"id"`
		Name string `json:"name"`
	}
	return Table{
		// middleware can write errors to nullify the request
		http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if req.Method != "POST" {
				resp.Header().Set("Allow", "POST")
				http.Error(resp, "only POST requests are allowed", http.StatusMethodNotAllowed)
			}
		}),

		// 'routes' just don't to respond to things they are uninterested in.
		http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/puppy") {
				return
			}
			id := atomic.AddInt64(&idcount, 1)
			json.NewEncoder(resp).Encode(Pet{id, "bowser"})
		}),
		http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if !strings.HasPrefix(req.URL.Path, "/kitty") {
				return
			}
			id := atomic.AddInt64(&idcount, 1)
			json.NewEncoder(resp).Encode(Pet{id, "meowser"})
		}),
	}
}

func ExampleSpy() {
	server := httptest.NewServer(MyService())
	defer server.Close()

	c := new(http.Client)
	c.Timeout = time.Second
	for i, r := range []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/", ""},
		{"POST", "/", ""},
		{"POST", "/kitty", "meow"},
	} {
		if i > 0 {
			fmt.Println()
			fmt.Println("===")
			fmt.Println()
		}
		requrl := server.URL + r.path
		var body io.Reader
		if r.body != "" {
			body = strings.NewReader(r.body)
		}
		req, err := http.NewRequest(r.method, requrl, body)
		if err != nil {
			fmt.Println(err)
			return
		}
		resp, err := c.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resp.Status, resp.Proto)
		io.Copy(os.Stdout, resp.Body)
		resp.Body.Close()
	}
	// Output:
	// 405 Method Not Allowed HTTP/1.1
	// only POST requests are allowed
	//
	// ===
	//
	// 404 Not Found HTTP/1.1
	// 404 page not found
	//
	// ===
	//
	// 200 OK HTTP/1.1
	// {"id":1,"name":"meowser"}
}
