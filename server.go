package main

import (
	"net/http"
	"net"
	"log"
	"embed"
	"io/fs"
)

//go:embed content/*
var fsys embed.FS
var c fs.FS
func init() {
	contentdir, err := fs.Sub(fsys,"content")
	if err != nil {
		log.Fatalln("failed to open content directory - does it exist?")
	}
	c = contentdir
}

func main() {
	l, err := net.Listen("tcp","0.0.0.0:80")
	if err != nil {
		log.Fatalf("failed to listen on TCP port 80 - are you root? Error:\n%s", err.Error())
	}
	log.Printf("now listening on TCP %s",l.Addr().String())
	http.HandleFunc("/api/", api)
	http.Handle("/",http.FileServerFS(c))
	http.Serve(l, nil)
}