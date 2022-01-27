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

const startUpFlag = "/tmp/httpserver_ready"

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
	// this code block is useless, just for testing reading config file
	config_path := os.Getenv("CONFIG_PATH")
	if config_path != "" {
		log.Printf("Got config path from env: %s", config_path)
	} else {
		config_path = "/etc/config/app.ini"
		log.Printf("env does not have CONFIG_PATH, use default %s", config_path)
	}
	configBytes, err := os.ReadFile(config_path)
	if err != nil {
		log.Fatalf("read configuration file failed: %v", err)
	}
	log.Printf("Loading configuration conent")
	log.Printf("%s = %s", config_path, string(configBytes))
	// Setup HTTP service
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
	// Output some info to indicate the service started
	log.Printf("Server started on %s", serverAddr)
	version := os.Getenv("VERSION")
	log.Printf("version=%s", version)

	// Create startup flag file
	log.Printf("Creating startup ready flag %s", startUpFlag)
	fp, err := os.Create(startUpFlag)
	defer fp.Close()
	if err != nil {
		log.Panicf("Failed to create startup flag, %v", err)
	}

	// wait for done channel receive signal
	sig := <-done
	log.Printf("Got signal %s", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer func() {
		log.Println("Running clean up...")
		log.Printf("Deleting start flag file %s", startUpFlag)
		if err := os.Remove(startUpFlag); err != nil {
			log.Println("Failed to clear start flag file, clean it manually")
		}
		srv.Close()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Failed to gracefully shutdown the http server: %+v", err)
	}
	log.Println("Server properly stopped")
}
