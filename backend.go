package main

import (
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

var catalogue [][]string
var noMatchResult result
var emptyQueryResult result
var headers map[string]int
type result map[string]string
var catPath *string
var debug bool
var portFlag int

func init() {
	// flags
	catPath = flag.String("cat","catalogue.csv","specify path to catalogue csv file")
	cc := flag.Int("cache", 5000, "set maximum cache capacity")
	d := flag.Bool("debug", false, "show additional debug info")
	p := flag.Int("port", 8080, "set tcp port to use")
	if !flag.Parsed() {
		flag.Parse()
	}
	portFlag = *p
	debug = *d
	debugPrint("debug logging enabled")
	cacheCapacity = *cc
	// load catalogue
	f, err := os.Open(*catPath)
	if err != nil {
		log.Fatalf("failed to open catalogue file %s with error\n%s",*catPath,err.Error())
	}
	catBytes, err := io.ReadAll(f)
	if err != nil {
		log.Fatalf("failed to read from catalogue %s with error\n%s",*catPath, err.Error())
	}
	catData := string(catBytes)
	r := csv.NewReader(strings.NewReader(catData))
	catalogue, err = r.ReadAll()
	if err != nil {
		log.Fatalln("failed to read catalogue csv - is it valid?")
	}
	// create header references
	headerRow := catalogue[0]
	headers = make(map[string]int)
	headers["title"] = slices.Index(headerRow, "title")
	headers["creators"] = slices.Index(headerRow,"creators")
	headers["date"] = slices.Index(headerRow,"publish_date")
	headers["tags"] = slices.Index(headerRow,"tags")
	headers["pages"] = slices.Index(headerRow,"length")
	headers["description"] = slices.Index(headerRow, "description")
	headers["copies"] = slices.Index(headerRow,"copies")
	for key, val := range headers {
		if val == -1 {
			log.Fatalf("didn't find required key \"%s\" in csv header",key)
		}
	}
	noMatchResult = make(result)
	noMatchResult["title"] = "No Match"
	noMatchResult["creators"] = "Try a different query."
	noMatchResult["date"] = ""
	noMatchResult["tags"] = ""
	noMatchResult["pages"] = ""
	noMatchResult["special"] = "1"

	emptyQueryResult = make(result)
	emptyQueryResult["title"] = "Empty Query"
	emptyQueryResult["creators"] = "Your search term cannot be blank."
	emptyQueryResult["date"] = ""
	emptyQueryResult["tags"] = ""
	emptyQueryResult["pages"] = ""
	emptyQueryResult["special"] = "1"
}

func find(q string, mode int) []result {
	debugPrint("find() called - got mode", mode, "and query", q)
	if q == "" {
		return []result{emptyQueryResult}
	}
	results := make([]result,0)
	for index, row := range catalogue {
		if index == 0 {
			continue
		}
		title := row[headers["title"]]
		authors := row[headers["creators"]]
		tags := row[headers["tags"]]
		pages := row[headers["pages"]]
		keywords := strings.Split(q, " ")
		m := 0
		for _, item := range keywords {
			switch {
			case mode == 0 && matchesAny(item, title, authors, tags):
				m++
			case mode == 1 && matches(title, item):
				m++
			case mode == 2 && matches(authors, item):
				m++
			case mode == 3 && matches(tags, item):
				m++
			}
		}
		if m == len(keywords) {
			new := make(result)
			new["title"] = title
			new["creators"] = authors
			new["date"] = row[headers["date"]]
			new["tags"] = tags
			new["pages"] = pages
			// -1 for the header row
			new["index"] = strconv.Itoa(index - 1)
			results = append(results, new)
		}
	}
	if len(results) == 0 {
		results = append(results, noMatchResult)
		return results
	}
	return results
}

func matches(metadata, query string) bool {
	return strings.Contains(strings.ToLower(metadata),strings.ToLower(query))
}

func matchesAny(query string, metadatas ...string) bool {
	for _, metadata := range metadatas {
		if strings.Contains(strings.ToLower(metadata),strings.ToLower(query)) {
			return true
		}
	}
	return false
}

func debugPrint(a ...any) {
	if debug {
		log.Println(a...)
	}
}