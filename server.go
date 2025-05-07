package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

//go:embed content/*
var fsys embed.FS
var c fs.FS
var port int = 80
var start time.Time

func handleInterrupt(channel chan os.Signal) {
	c := <- channel
	fmt.Print("\r")
	log.Printf("Received %s", c.String())
	views := 0.0
	detailsViewed := 0.0
	for _, item := range tokens {
		views += float64(item.Views)
		if item.Views > 0 {
			detailsViewed++
		}
	}
	if len(tokens) == 0 {
		log.Printf("Statistics:\nRan for %s, and processed %d API requests.\nUsers searched %d times, and viewed details of items %d times.\nUnfortunately, detailed statistics couldn't be generated because no tokens were issued.",time.Since(start).Round(time.Second).String(), counters.Requests, counters.Searches, counters.ItemDetails)
		os.Exit(0)
	}
	log.Printf(
		"Statistics:\nRan for %s, and processed %d API requests.\nUsers searched %d times, and viewed details of items %d times.\nOn average, users viewed %s items per search.\n%d%% of people viewed item details after searching.\nThank you for using Phosphor!",
		time.Since(start).Round(time.Second).String(),
		counters.Requests,
		counters.Searches,
		counters.ItemDetails,
		strconv.FormatFloat(views / float64(len(tokens)),'f',2,64),
		int(math.Round(100 * (detailsViewed / float64(len(tokens))))),
	)
	os.Exit(0)
}

func init() {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)
	go handleInterrupt(channel)
	contentdir, err := fs.Sub(fsys, "content")
	if err != nil {
		log.Fatalln("failed to open content directory - does it exist?")
	}
	c = contentdir
}

func main() {
	// if the home directory is /root, use port 80
	// as the program is running as superuser
	if h, err := os.UserHomeDir(); err == nil && strings.HasPrefix(h, "/root") {
		port = 80
	} else if err == nil && strings.HasPrefix(h, "/home") {
		port = 8080
	} else {
		port = 80
	}
	// if the port has been specified manually
	// use that instead (default is 8080)
	if portFlag != 8080 {
		port = portFlag
	}
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen on TCP port %d - are you root? Error:\n%s", port, err.Error())
	}
	log.Printf("now listening on TCP %s", l.Addr().String())
	http.HandleFunc("/dynamic/", api)
	http.Handle("/", http.FileServerFS(c))
	start = time.Now()
	http.Serve(l, nil)
}
