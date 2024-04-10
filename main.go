package main

import (
	"flag"
	"log"
	"net/http"
	"sharded-kvs/db"
	"sharded-kvs/web"
)

/*
 * List of flags, basically important constants used by the code!
 */
var (
	dbLocation = flag.String("db-location", "", "Path to the Bolt DB")
	httpAddr   = flag.String("http-addr", "127.0.0.1:8080", "HTTP Host and Port")
)

/*
 * Validation of flags!
 */
func parseFlags() {
	flag.Parse()
	if *dbLocation == "" {
		log.Fatalf("DB-Location must be provided!")
	}
}

func main() {
	parseFlags()
	db, close, err := db.NewDatabase(*dbLocation)
	if err != nil {
		log.Fatalf("NewDatabase(%q): %v", *dbLocation, err)
	}
	defer close()

	srv := web.NewServer(db)
	http.HandleFunc("/get", srv.GetHandler)
	http.HandleFunc("/set", srv.SetHandler)

	// no need to parse httpAddr since it ain't empty!
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
