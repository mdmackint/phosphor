package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

//go:embed content/*
var fsys embed.FS
var c fs.FS
var port int = 80

func init() {
	if h, err := os.UserHomeDir(); err == nil && strings.HasPrefix(h, "/root") {
		port = 80
	} else if err == nil && strings.HasPrefix(h, "/home") {
		port = 8080
	} else {
		port = 80
	}
	contentdir, err := fs.Sub(fsys, "content")
	if err != nil {
		log.Fatalln("failed to open content directory - does it exist?")
	}
	c = contentdir
}

func main() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen on TCP port %d - are you root? Error:\n%s", port, err.Error())
	}
	log.Printf("now listening on TCP %s", l.Addr().String())
	http.HandleFunc("/dynamic/", api)
	http.Handle("/", http.FileServerFS(c))
	http.Serve(l, nil)
}
