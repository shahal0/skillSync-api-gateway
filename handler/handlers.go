package handler

import (
	"io"
	"log"
	"net/http"
	"os"
)

var (
	AuthServiceURL = os.Getenv("AUTH_SERVICE_URL")
	JobServiceURL  = os.Getenv("JOB_SERVICE_URL")
)

// ProxyRequest forwards the request to the target service
func ProxyRequest(w http.ResponseWriter, r *http.Request, target string) {
	req, err := http.NewRequest(r.Method, target+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	req.Header = r.Header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// AuthHandler handles authentication-related requests
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	if AuthServiceURL == "" {
		log.Println("AUTH_SERVICE_URL not set")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	ProxyRequest(w, r, AuthServiceURL)
}

// JobHandler handles job-related requests
func JobHandler(w http.ResponseWriter, r *http.Request) {
	if JobServiceURL == "" {
		log.Println("JOB_SERVICE_URL not set")
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	ProxyRequest(w, r, JobServiceURL)
}
