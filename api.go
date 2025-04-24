package main

import (
	"embed"
	"html"
	"log"
	"net/http"
	"strings"
)

//go:embed internal
var internals embed.FS

func api(w http.ResponseWriter, r *http.Request) {
	// get data from post request, log it, then search the db
	search := r.FormValue("enquiry")
	log.Printf("new api request from ip %s",r.RemoteAddr)
	res := find(search)
	log.Printf("got %d results", len(res))

	// read the html file, respond with HTTP 500 if file fails to load
	respBytes, err := internals.ReadFile("internal/apireturn.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := string(respBytes)

	// write list items to string
	lis := ""
	if len(res) == 0 {
		w.WriteHeader(500)
		return
	}
	for _, item := range res {
		li := newLi(item)
		lis += li
	}
	
	// make replacements in the html
	// to add the results and query (escaped)
	resp = strings.ReplaceAll(resp,"{ %%% }",lis)
	resp = strings.ReplaceAll(resp,"{ query }",html.EscapeString(search))

	// write response
	w.WriteHeader(200)
	_, err = w.Write([]byte(resp))
	if err != nil {
		log.Println("Failed to write bytes in request!")
		w.WriteHeader(500)
		return
	}
	return
}

// create list element from result
func newLi(r result) (li string) {
	li = "<li><b>{t}</b><br><i>{a}, {d}</i></li>"
	li = strings.ReplaceAll(li,"{t}",html.EscapeString(r["title"]))
	li = strings.ReplaceAll(li,"{a}",html.EscapeString(r["creators"]))
	li = strings.ReplaceAll(li, "{d}",html.EscapeString(r["date"]))
	return
}