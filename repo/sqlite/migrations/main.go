package main

import (
	"database/sql"
	"flag"
	"log"

	_ "modernc.org/sqlite"

	sqlitedb "restful-boilerplate/repo/sqlite"
)

func main() {
	dbPath := flag.String("db", "./data.db", "path to sqlite database file")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close() //nolint:errcheck // best-effort close on exit

	if err := sqlitedb.Migrate(db); err != nil {
		log.Fatalf("apply schema: %v", err)
	}
	log.Println("migration applied successfully")
}
