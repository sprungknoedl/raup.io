package main

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type MediaHandler func(w http.ResponseWriter, r *http.Request, dst *url.URL, resp *http.Response) error

var MediaHandlers = map[string]MediaHandler{
	"text/html": handleHTML,
	//TODO: handle css url()
}

func handleHTML(w http.ResponseWriter, r *http.Request, dst *url.URL, resp *http.Response) error {
	// parse html response
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	var f func(*html.Node)
	f = func(node *html.Node) {

		if node.Type == html.ElementNode {
			for i, attr := range node.Attr {

				// patch links
				if attr.Key == "href" || attr.Key == "src" || attr.Key == "action" {
					href, _ := dst.Parse(attr.Val)
					node.Attr[i].Val = "/" + strings.Replace(href.String(), "://", "/", 1)
				}

				// remove javascript event handler
				if strings.HasPrefix(attr.Key, "on") {
					node.Attr[i].Val = ""
				}

				// remove flash and similiar objects
				if attr.Key == "data" {
					node.Attr[i].Val = ""
				}

			}
		}

		// remove script tags as a whole
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode && child.Data == "script" {
				prev := child.PrevSibling
				node.RemoveChild(child)
				child = prev
			}

			f(child)
		}
	}
	f(doc)

	// send patched html to client
	html.Render(w, doc)
	return nil
}
