package main

import (
	"log"
	"net/http"

	"github.com/UKHomeOffice/snowsync/pkg/in"
	"github.com/apex/gateway"
)

func main() {

	http.HandleFunc("/v2/in", in.HandleNew)
	http.HandleFunc("/v2/add", in.HandleAdd)
	log.Fatal(gateway.ListenAndServe("", nil))
}
