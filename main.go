package main

import (
	"excel_parser/database"
	"excel_parser/starter"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		_ = http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	defer database.Close()

	if err := starter.Start(); err != nil {
		panic(err)
	}

}
