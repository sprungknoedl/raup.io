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

func IndexR(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/@about", http.StatusMovedPermanently)
		return
	}

	if r.URL.Path == "/favicon.ico" {
		http.Redirect(w, r, "/static/favicon.ico", http.StatusMovedPermanently)
		return
	}

	http.Redirect(w, r, "/http/"+r.URL.Path, http.StatusMovedPermanently)
}

func RedirectR(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	dst := r.PostForm.Get("url")
	http.Redirect(w, r, toPath(dst), http.StatusSeeOther)
}

func AboutR(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Name string
	}{
		Name: "raup.io",
	}
	tpl.ExecuteTemplate(w, "index.html", data)
}

func ProxyR(w http.ResponseWriter, r *http.Request) {
	rawurl := toURL(r.URL.Path)

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
	mux.HandleFunc("/redirect", RedirectR)
	mux.HandleFunc("/@about", AboutR)
	mux.HandleFunc("/http/", ProxyR)
	mux.HandleFunc("/https/", ProxyR)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	logger := handlers.LoggingHandler(os.Stdout, mux)
	http.ListenAndServe(":7777", logger)
}
