package main

import (
	"embed"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type cacheItem struct {
	Res   []result
	Query string
}

//go:embed internal
var internals embed.FS
var cache []cacheItem

// Change this to suit your requirements
var cacheCapacity int = 5000

func init() {
	cache = make([]cacheItem, 0, cacheCapacity)
}

func api(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/search", "api/search":
		searchHandler(w,r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	// get data from post request, log it, then search the db
	search := r.FormValue("enquiry")
	log.Printf("new api request from ip %s", r.RemoteAddr)
	start := time.Now()
	var res []result
	if len(cache) > 0 {
		for _, item := range cache {
			if item.Query == strings.Trim(search, " ") {
				res = item.Res
				log.Printf("cache used for query \"%s\"", html.EscapeString(search))
				break
			}
		}
		if res == nil {
			res = find(search)
			switch {
			case len(cache) < cacheCapacity:
				cache = append(cache, cacheItem{Res: res, Query: strings.Trim(search, " ")})
				log.Println("cache appended")
			case len(cache) >= cacheCapacity:
				cache = make([]cacheItem, 0, cacheCapacity)
				log.Println("cache cleared")
			}
		}
	} else {
		res = find(search)
		cache = append(cache, cacheItem{Res: res, Query: strings.Trim(search, " ")})
		log.Println("cache appended")
	}
	dur := time.Since(start)
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
	p := message.NewPrinter(language.English)
	resp = strings.ReplaceAll(resp, "{ %%% }", lis)
	resp = strings.ReplaceAll(resp, "{ query }", html.EscapeString(search))
	resp = strings.ReplaceAll(resp, "{ dur }", p.Sprintf("%dμs", dur.Microseconds()))

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
	li = "<li><b>{t}</b><br><i>{a}{d}{tags}{pages}</i></li>"
	li = strings.ReplaceAll(li, "{t}", html.EscapeString(html.UnescapeString(r["title"])))
	li = strings.ReplaceAll(li, "{a}", html.EscapeString(html.UnescapeString(r["creators"])))
	if len(r["date"]) != 0 {
		li = strings.ReplaceAll(li, "{d}", "<br>Published "+html.EscapeString(html.UnescapeString(r["date"])))
	} else {
		li = strings.ReplaceAll(li, "{d}", "")
	}
	if len(r["tags"]) == 0 {
		li = strings.ReplaceAll(li, "{tags}", "")
	} else {
		li = strings.ReplaceAll(li, "{tags}", "<br>Tags: "+r["tags"])
	}
	if len(r["pages"]) == 0 {
		li = strings.ReplaceAll(li, "{pages}", "")
	} else {
		li = strings.ReplaceAll(li, "{pages}", fmt.Sprintf("<br>%s pages", r["pages"]))
	}
	return
}
