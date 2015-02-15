package main

import (
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/handlers"
)

var client = &http.Client{}
var tpl = template.Must(template.ParseGlob("html/*.html"))

func main() {
	mux := http.NewServeMux()
	// webui.go
	mux.HandleFunc("/", IndexR)
	mux.HandleFunc("/redirect", RedirectR)
	mux.HandleFunc("/@about", AboutR)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// main.go
	mux.HandleFunc("/http/", ProxyR)
	mux.HandleFunc("/https/", ProxyR)

	// is request logging valid for an anonymous service?
	// TODO: remove in production deployment
	logger := handlers.LoggingHandler(os.Stdout, mux)
	http.ListenAndServe(":7777", logger)
}

func ProxyR(w http.ResponseWriter, r *http.Request) {
	// parse request as target url
	dst, err := url.Parse(toURL(r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	// build request
	req, err := http.NewRequest(r.Method, dst.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	// copy client headers
	CopyHeader(req.Header, r.Header, "Accept", Id)
	CopyHeader(req.Header, r.Header, "Accept-Charset", Id)
	CopyHeader(req.Header, r.Header, "Accept-Language", Id)
	CopyHeader(req.Header, r.Header, "Content-Type", Id)

	// get target
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	// UP: client request
	// ----
	// DOWN: server response

	//copy server headers
	CopyHeader(w.Header(), resp.Header, "Content-Type", Id)
	CopyHeader(w.Header(), resp.Header, "Location", func(header string) string {
		// rewrite redirect location header
		location, _ := dst.Parse(header)
		return location.String()
	})

	// wrap response writer and set status code
	//w = NewCodeResponseWriter(resp.StatusCode, w)

	// copy/transform body
	typ := resp.Header.Get("Content-Type")
	mediatype, _, _ := mime.ParseMediaType(typ)
	if handler, ok := MediaHandlers[mediatype]; ok {

		// Let handler transform the body. The function should make sure
		// that it returns eventual errors before it writes to the
		// ResponseWriter.
		if err := handler(w, r, dst, resp); err != nil {
			http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		}

	} else {
		// no html, so pass through
		io.Copy(w, resp.Body)
	}
}

func Id(header string) string { return header }

func CopyHeader(dst http.Header, src http.Header, name string, fn func(string) string) {
	cname := http.CanonicalHeaderKey(name)
	header := src.Get(cname)
	if header != "" {
		dst.Add(cname, fn(header))
	}
}

type CodeResponseWriter struct {
	Code        int
	wroteHeader bool
	writer      http.ResponseWriter
}

func NewCodeResponseWriter(code int, w http.ResponseWriter) *CodeResponseWriter {
	return &CodeResponseWriter{
		Code:        code,
		wroteHeader: false,
		writer:      w,
	}
}

func (c *CodeResponseWriter) Header() http.Header {
	return c.writer.Header()
}

func (c *CodeResponseWriter) Write(data []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(c.Code)
	}

	return c.writer.Write(data)
}

func (c *CodeResponseWriter) WriteHeader(code int) {
	c.Code = code
	c.wroteHeader = true
	c.writer.WriteHeader(code)
}

func toURL(rawurl string) string {
	rawurl = strings.Replace(rawurl, "/http/", "http://", 1)
	rawurl = strings.Replace(rawurl, "/https/", "https://", 1)
	return rawurl
}

func toPath(rawurl string) string {
	rawurl = strings.Replace(rawurl, "http://", "/http/", 1)
	rawurl = strings.Replace(rawurl, "https://", "/https/", 1)
	return rawurl
}
