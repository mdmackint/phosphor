package main

import (
	"encoding/csv"
	"flag"
	"io"
	"log"
	"os"
	"slices"
	"strings"
)

var catalogue [][]string
var noMatchResult result
var emptyQueryResult result
var headers map[string]int
type result map[string]string
var catPath *string

func init() {
	catPath = flag.String("cat","catalogue.csv","specify path to catalogue csv file")
	if !flag.Parsed() {
		flag.Parse()
	}
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
	headerRow := catalogue[0]
	headers = make(map[string]int)
	headers["title"] = slices.Index(headerRow, "title")
	headers["creators"] = slices.Index(headerRow,"creators")
	headers["date"] = slices.Index(headerRow,"publish_date")
	headers["tags"] = slices.Index(headerRow,"tags")
	headers["pages"] = slices.Index(headerRow,"length")
	for key, val := range headers {
		if val == -1 {
			log.Fatalf("didn't find required key %s in csv header",key)
		}
	}
	noMatchResult = make(result)
	noMatchResult["title"] = "No Match"
	noMatchResult["creators"] = "Try a different query."
	noMatchResult["date"] = ""
	noMatchResult["tags"] = ""
	noMatchResult["pages"] = ""

	emptyQueryResult = make(result)
	emptyQueryResult["title"] = "Empty Query"
	emptyQueryResult["creators"] = "Your search term cannot be blank."
	emptyQueryResult["date"] = ""
	emptyQueryResult["tags"] = ""
	emptyQueryResult["pages"] = ""
}

func find(q string) []result {
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
			if matches(title, item) || matches(authors, item) || matches(tags, item) {
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
			results = append(results, new)
		}
	}
	if len(results) == 0 {
		results = append(results, noMatchResult)
		return results
	}
	return results
}

func matches(title, query string) bool {
	return strings.Contains(strings.ToLower(title),strings.ToLower(query))
}