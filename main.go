package main

import (
	"excel_parser/starter"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	if err := starter.Start(); err != nil {
		panic(err)
	}
}
