package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"golang.org/x/net/html"
)

var tpl = template.Must(template.ParseGlob("html/*.html"))
var blacklist = map[string]bool{
	"/favicon.ico": false,
	"/robots.txt":  false,
}

func IndexR(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/@about", http.StatusMovedPermanently)
		return
	}

	if _, ok := blacklist[r.URL.Path]; ok {
		http.Error(w, r.URL.Path+" not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/http/"+r.URL.Path, http.StatusMovedPermanently)
}

func AboutR(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Name string
	}{
		Name: "pro.xy",
	}
	tpl.ExecuteTemplate(w, "index.html", data)
}

func ProxyR(w http.ResponseWriter, r *http.Request) {
	rawurl := r.URL.Path
	rawurl = strings.Replace(rawurl, "/http/", "http://", 1)
	rawurl = strings.Replace(rawurl, "/https/", "https://", 1)

	// parse request as target url
	dst, err := url.Parse(rawurl)
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	// get target
	resp, err := http.Get(dst.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%s: %d | %+v\n", dst, resp.StatusCode, resp.Header)
	typ := resp.Header.Get("Content-Type")
	w.Header().Add("Content-Type", typ)

	if strings.Contains(typ, "text/html") {
		PatchHtml(w, r, dst, resp)
		return
	}

	// no html, so pass through
	io.Copy(w, resp.Body)
}

func PatchHtml(w http.ResponseWriter, r *http.Request, dst *url.URL, resp *http.Response) {
	// parse html response
	doc, err := html.Parse(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("err: %v\n", err), http.StatusInternalServerError)
		return
	}

	// patch links
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for i, attr := range n.Attr {
				if attr.Key == "href" || attr.Key == "src" {
					href, _ := dst.Parse(attr.Val)
					n.Attr[i].Val = "/" + strings.Replace(href.String(), "://", "/", 1)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// send patched html to client
	html.Render(w, doc)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", IndexR)
	mux.HandleFunc("/@about", AboutR)
	mux.HandleFunc("/http/", ProxyR)
	mux.HandleFunc("/https/", ProxyR)

	logger := handlers.LoggingHandler(os.Stdout, mux)
	http.ListenAndServe(":7777", logger)
}
