package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func main() {
	demoUrl, err := url.Parse("http://0.0.0.0:8000")
	if err != nil {
		log.Fatal(err)
	}
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = demoUrl.Host
		r.URL.Host = demoUrl.Host
		r.URL.Scheme = demoUrl.Scheme
		// r.URL.Opaque = demoUrl.Opaque
		r.RequestURI = ""
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("error occured when requesting client:\n%v\n%v\n", w, err)
		}

		// copy res headers
		fmt.Printf("headers: %v\n", res.Header)
		for key, value := range res.Header {
			for _, v := range value {
				w.Header().Set(key, v)
			}
		}
		forwardedFor, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			fmt.Printf("error splitting HostPort: %v", err)
		}

		// Handle streaming
		done := make(chan bool)
		go func() {
			for {
				select {
				case <-time.Tick(10 * time.Millisecond):
					w.(http.Flusher).Flush()
				case <-done:
					return
				}
			}
		}()

		// handle trailer (trailer headers)
		trailerKeys := []string{}
		for key := range r.Trailer {
			trailerKeys = append(trailerKeys, key)
		}

		w.Header().Set("Trailer", strings.Join(trailerKeys, ","))

		// add sender ip to x-forwarded-for header
		w.Header().Set("X-FORWARDED-FOR", forwardedFor)

		w.WriteHeader(res.StatusCode)
		// copy res body
		io.Copy(w, res.Body)
		// fill trailer headers
		for keys, values := range r.Trailer {
			for _, value := range values {
				w.Header().Set(keys, value)
			}
		}
		// to stop streaming when done
		close(done)
	})
	addr := ":8080"
	http.ListenAndServe(addr, proxy)
}
