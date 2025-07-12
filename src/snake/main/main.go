package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"snake/src/snake/connection" // <-- Update if you move the code up a level
)

func mainHandler(w http.ResponseWriter, req *http.Request) {
	file, err := os.Open("index.html")
	if err == nil {
		defer file.Close()
		io.Copy(w, file)
	} else {
		http.Error(w, "index.html not found", http.StatusNotFound)
	}
}

func main() {
	portPtr := flag.Int("port", 8000, "server port")
	flag.Parse()

	http.HandleFunc("/", mainHandler)

	// Serve static files from ./static/ directory
	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))
	http.Handle("/static/", fs)

	// WebSocket endpoint (updated for Gorilla)
	http.HandleFunc("/ws", connection.ConnectionHandler)

	err := http.ListenAndServe(fmt.Sprintf(":%d", *portPtr), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
