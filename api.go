package main

import (
	"crypto/rand"
	"embed"
	"fmt"
	"html"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type cacheItem struct {
	Res   []result
	Query string
	Mode  int
}

type token struct {
	Token  string
	Issued time.Time
	Query  string
	Views  int
}

type countersStruct struct {
	Requests uint64
	Searches uint64
	ItemDetails uint64
}

//go:embed internal
var internals embed.FS
var cache []cacheItem
var cacheCapacity int = 5000
var genericErrorPage string
var errorPages map[int][]byte
var tokens map[string]*token
var counters countersStruct

func newErrorPage(code int, details string) (page []byte) {
	str := genericErrorPage
	str = strings.ReplaceAll(str, "{ code }", fmt.Sprintf("%d - %s", code, http.StatusText(code)))
	str = strings.ReplaceAll(str, "{ details }", details)
	page = []byte(str)
	return
}

func init() {
	// make cache and token map
	cache = make([]cacheItem, 0, cacheCapacity)
	tokens = make(map[string]*token)

	// make errorPages map
	errorPages = make(map[int][]byte)
	// load generic error page
	errorPageBytes, err := internals.ReadFile("internal/error.html")
	if err != nil {
		log.Fatalln("unable to read file internal/error.html\n", err.Error())
	}
	genericErrorPage = string(errorPageBytes)

	// create error pages
	errorPages[http.StatusNotFound] = newErrorPage(http.StatusNotFound, "This page wasn't found. Please ensure that you entered a valid path.")
	errorPages[http.StatusInternalServerError] = newErrorPage(
		http.StatusInternalServerError,
		"The server experienced an internal error while trying to handle your request. This is likely temporary, and is not a problem with your computer.",
	)
	errorPages[http.StatusBadRequest] = newErrorPage(
		http.StatusBadRequest,
		"You've performed a malformed request. Please ensure any form data is entered correctly.",
	)
}

func api(w http.ResponseWriter, r *http.Request) {
	counters.Requests++
	switch r.URL.Path {
	case "/dynamic/search", "dynamic/search":
		searchHandler(w, r)
	case "/dynamic/details", "dynamic/details":
		detailsHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write(errorPages[http.StatusNotFound])
		return
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	// get data from post request, log it, then search the db
	search := r.FormValue("enquiry")
	if len(search) > 512 {
		search = search[:512]
	}
	var mode int
	switch r.FormValue("fields") {
	case "all", "":
		mode = 0
	case "title":
		mode = 1
	case "creator":
		mode = 2
	case "tags":
		mode = 3
	default:
		mode = 0
	}
	log.Printf("new api request from ip %s", r.RemoteAddr)
	start := time.Now()
	var res []result
	if len(cache) > 0 {
		for _, item := range cache {
			if item.Query == strings.Trim(search, " ") && item.Mode == mode {
				res = item.Res
				log.Printf("cache used for query \"%s\"", clean(search))
				break
			}
		}
		if res == nil {
			res = find(search, mode)
			switch {
			case len(cache) < cacheCapacity:
				cache = append(cache, cacheItem{Res: res, Query: strings.Trim(search, " "), Mode: mode})
				log.Println("cache appended")
			case len(cache) >= cacheCapacity:
				cache = make([]cacheItem, 0, cacheCapacity)
				log.Println("cache cleared")
			}
		}
	} else {
		res = find(search, mode)
		cache = append(cache, cacheItem{Res: res, Query: strings.Trim(search, " "), Mode: mode})
		log.Println("cache appended")
	}
	dur := time.Since(start)
	if len(res) == 1 && res[0]["special"] == "1" {
		log.Println("got either no results or an invalid query")
	} else {
		log.Printf("got %d results", len(res))
	}

	// read the html file, respond with HTTP 500 if file fails to load
	respBytes, err := internals.ReadFile("internal/apireturn.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorPages[http.StatusInternalServerError])
		return
	}
	resp := string(respBytes)

	// write list items to string
	lis := ""

	// there should always be at least 1 result, because if nothing is found
	// then find() returns "no results" as a result
	if len(res) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorPages[http.StatusInternalServerError])
		return
	}

	t := newToken()
	t.Query = search
	tokens[t.Token] = t
	for _, item := range res {
		li := newLi(item, t)
		lis += li
	}

	// make replacements in the html
	// to add the results and query (escaped)
	p := message.NewPrinter(language.English)
	resp = strings.ReplaceAll(resp, "{ %%% }", lis)
	resp = strings.ReplaceAll(resp, "{ query }", html.EscapeString(search))
	resp = strings.ReplaceAll(resp, "{ dur }", p.Sprintf("%dμs", dur.Microseconds()))
	switch mode {
	case 0:
		resp = strings.NewReplacer("{ everything }", " selected", "{ title }", "", "{ author }", "", "{ tags }", "").Replace(resp)
	case 1:
		resp = strings.NewReplacer("{ everything }", "", "{ title }", " selected", "{ author }", "", "{ tags }", "").Replace(resp)
	case 2:
		resp = strings.NewReplacer("{ everything }", "", "{ title }", "", "{ author }", " selected", "{ tags }", "").Replace(resp)
	case 3:
		resp = strings.NewReplacer("{ everything }", "", "{ title }", "", "{ author }", "", "{ tags }", " selected").Replace(resp)
	}

	// write response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(resp))
	if err != nil {
		log.Println("Failed to write bytes in request!")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorPages[http.StatusInternalServerError])
		return
	}
	counters.Searches++
	return
}

func detailsHandler(w http.ResponseWriter, r *http.Request) {
	// get index to lookup from form value
	indexValue := r.FormValue("i")
	tString := r.FormValue("t")
	var t *token
	if token, ok := tokens[tString]; ok {
		t = token
		t.Views++
	} else {
		t = nil
	}
	// check for blank index, and redirect user home
	if indexValue == "" {
		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// convert index to integer
	var (
		index int
		err   error
	)
	if index, err = strconv.Atoi(indexValue); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorPages[http.StatusBadRequest])
		return
	}

	// get details page
	page, err := internals.ReadFile("internal/details.html")
	if err != nil {
		log.Printf("couldn't read details page! returning http 500. error:\n%s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(errorPages[http.StatusInternalServerError])
		return
	}

	// get catalogue entry
	var entry result = make(result, 6)
	// adding 1 to index because of header row
	index++
	// ensure there isn't a reference error.
	// it's not a mistake that it's checking if index < 1 and not < 0
	// because we don't want the header row to be returned
	if index >= len(catalogue) || index < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(errorPages[http.StatusBadRequest])
		return
	}
	// fetch details of entry
	entry["title"] = catalogue[index][headers["title"]]
	entry["creators"] = catalogue[index][headers["creators"]]
	entry["description"] = catalogue[index][headers["description"]]
	entry["date"] = catalogue[index][headers["date"]]
	entry["pages"] = catalogue[index][headers["pages"]]
	entry["tags"] = catalogue[index][headers["tags"]]
	entry["copies"] = catalogue[index][headers["copies"]]
	debugPrint("details of item", entry["title"], "viewed")
	pageStr := string(page)
	pageStr = strings.ReplaceAll(pageStr, "{ title }", entry["title"])
	// remove placeholders of empty values
	for i := range 4 {
		key := ""
		placeholder := ""
		switch i {
		case 0:
			key = "creators"
			placeholder = "{ author }"
		case 1:
			key = "date"
			placeholder = "{ date }"
		case 2:
			key = "tags"
			placeholder = "{ tags }"
		case 3:
			key = "description"
			placeholder = "{ desc }"
		}
		if len(strings.TrimSpace(entry[key])) == 0 {
			pageStr = strings.ReplaceAll(pageStr, placeholder, "")
		}
	}
	pageStr = strings.ReplaceAll(pageStr, "{ author }", fmt.Sprintf("by <b>%s</b>", clean(entry["creators"])))
	pageStr = strings.ReplaceAll(pageStr, "{ date }", fmt.Sprintf("<br>published %s", clean(entry["date"])))
	pageStr = strings.ReplaceAll(pageStr, "{ tags }", fmt.Sprintf("<br>tagged with %s", clean(entry["tags"])))
	pageStr = strings.ReplaceAll(pageStr, "{ desc }", fmt.Sprintf("<br>Description:<br><blockquote>%s</blockquote>", clean(entry["description"])))
	if t != nil {
		pageStr = strings.ReplaceAll(pageStr, "{ back }", fmt.Sprintf("<a href=\"javascript:history.back()\">back to your search for \"%s\"</a>", clean(t.Query)))
	} else {
		pageStr = strings.ReplaceAll(pageStr, "{ back }", "<a href=\"/\">back home</a>")
	}
	var copiesStr string
	switch entry["copies"] {
	case "1":
		copiesStr = "1 copy"
	default:
		copiesStr = clean(entry["copies"]) + " copies"
	}
	pageStr = strings.ReplaceAll(pageStr, "{ copy }", fmt.Sprintf("<br>%s", copiesStr))
	w.WriteHeader(200)
	w.Write([]byte(pageStr))
	counters.ItemDetails++
	return
}

// create list element from result
func newLi(r result, t *token) (li string) {
	li = "<li><b>{t}</b><br><i>{a}{d}{tags}{pages}</i></li>"
	index, err := strconv.Atoi(r["index"])
	var title string
	if err != nil {
		title = clean(r["title"])
	} else {
		title = fmt.Sprintf("<a href=\"/dynamic/details?i=%d&t=%s\">%s</a>", index, t.Token, clean(r["title"]))
	}
	li = strings.ReplaceAll(li, "{t}", title)
	li = strings.ReplaceAll(li, "{a}", clean(r["creators"]))
	if len(strings.TrimSpace(r["date"])) == 0 {
		li = strings.ReplaceAll(li, "{d}", "")
	} else {
		li = strings.ReplaceAll(li, "{d}", "<br>Published "+clean(r["date"]))
	}
	if len(strings.TrimSpace(r["tags"])) == 0 {
		li = strings.ReplaceAll(li, "{tags}", "")
	} else {
		li = strings.ReplaceAll(li, "{tags}", "<br>Tags: "+clean(r["tags"]))
	}
	if len(strings.TrimSpace(r["pages"])) == 0 {
		li = strings.ReplaceAll(li, "{pages}", "")
	} else {
		li = strings.ReplaceAll(li, "{pages}", fmt.Sprintf("<br>%s pages", clean(r["pages"])))
	}
	return
}

// unescape and re-escape string
func clean(s string) (c string) {
	c = html.EscapeString(html.UnescapeString(s))
	return
}

func newToken() *token {
	return &token{Issued: time.Now(), Token: rand.Text()}
}
