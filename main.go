package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
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
			fmt.Printf("key: %v value: %v\n", key, value)
			w.Header().Set(key, value[0])
		}
		forwardedFor, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			fmt.Printf("error splitting HostPort: ", err)
		}

		w.Header().Set("X-FORWARDED-FOR", forwardedFor)
		// copy res body
		w.WriteHeader(res.StatusCode)
		io.Copy(w, res.Body)
	})
	addr := ":8080"
	http.ListenAndServe(addr, proxy)
}
