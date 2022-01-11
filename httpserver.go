package main

import (
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
)

type Response struct {
	StatusCode int
	Data       string
}

func (r Response) make(rw http.ResponseWriter, req *http.Request) {
	log.Printf("Received %s %s %s from %s", req.Proto, req.Method, req.URL, req.RemoteAddr)
	log.Println("Reading VERSION from environment")
	version := os.Getenv("VERSION")
	log.Printf("VERSION=%s, adding to response header", version)
	rw.Header().Add("version", version)
	for field, value := range req.Header {
		for _, v := range value {
			log.Printf("setting response header: %s = %s", field, v)
			rw.Header().Add(field, v)
		}
	}
	log.Printf("Sending response to %s, status_code=%d", req.RemoteAddr, r.StatusCode)
	rw.WriteHeader(r.StatusCode)
	io.WriteString(rw, r.Data)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Calling healthz handler")
	response := Response{StatusCode: 200, Data: "ok"}
	response.make(w, r)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Calling default handler")
	response := Response{StatusCode: 200, Data: "welcome"}
	response.make(w, r)
}

func main() {
	mux := http.NewServeMux()
	debug := os.Getenv("HTTP_DEBUG")
	if debug == "1" {
		log.Println("Adding debug handlers...")
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/", defaultHandler)
	log.Println("Starting http server")
	log.Fatal(http.ListenAndServe(":8000", mux))
}
