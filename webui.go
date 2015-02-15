package main

import "net/http"

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
