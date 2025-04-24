package main

import (
	_ "embed"
	"encoding/csv"
	"log"
	"strings"
)

//go:embed catalogue.csv
var catData string
var catalogue [][]string

type result map[string]string

func init() {
	r := csv.NewReader(strings.NewReader(catData))
	var err error
	catalogue, err = r.ReadAll()
	if err != nil {
		log.Fatalln("failed to read catalogue csv - is it valid?")
	}
}

func find(q string) []result {
	results := make([]result,0)
	for index, row := range catalogue {
		if index == 0 {
			continue
		}
		title := row[1]
		authors := row[2]
		if matches(title, q) || matches(authors, q) {
			new := make(result)
			new["title"] = title
			new["creators"] = authors
			new["date"] = row[9]
			results = append(results, new)
		}
	}
	if len(results) == 0 {
		results[0]["title"] = "No Match"
		results[0]["creators"] = "Try a different query."
		return results
	}
	return results
}

func matches(title, query string) bool {
	return strings.Contains(strings.ToLower(title),strings.ToLower(query))
}