package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	serverAddr := ":8000"

	srv := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	log.Println("Starting http server")
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start http server, %+v", err)
		}
	}()

	log.Printf("Server started on %s", serverAddr)

	// wait for done channel receive signal
	sig := <-done
	log.Printf("Got signal %s", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer func() {
		log.Println("Running clean up...")
		srv.Close()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Failed to gracefully shutdown the http server: %+v", err)
	}
	log.Println("Server properly stopped")
}
