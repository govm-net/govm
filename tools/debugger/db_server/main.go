package main

import (
	"fmt"
	"github.com/lengzhao/govm/database"
	"net/http"
	"net/rpc"
	"os"
	"time"
	"flag"
)

func main() {
	reset := flag.Bool("reset",true,"reset database")
	address := flag.String("l","127.0.0.1:12345","listen address of db server")
	flag.Parse()
	if *reset{
		os.RemoveAll("db_dir")
	}
	
	db := database.TDb{}
	db.Init()
	defer db.Close()
	sr := rpc.NewServer()
	sr.Register(&db)

	fmt.Println("db server start linsten:", *address)
	server := &http.Server{
		Addr:         *address,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      sr,
	}
	server.ListenAndServe()
	server.Close()
}
