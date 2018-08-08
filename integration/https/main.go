package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("ok"))
		fmt.Println("ok")
	})

	fmt.Println(http.ListenAndServeTLS(":8943", "/Users/nicj/Developer/go/src/github.com/nicholasjackson/consul-connect-router/integration/https/cert.pem", "/Users/nicj/Developer/go/src/github.com/nicholasjackson/consul-connect-router/integration/https/key.pem", nil))
}
