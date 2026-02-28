package main

import (
	"context"
	"database/sql"
	"flag"
	"log"

	_ "modernc.org/sqlite"

	infradb "restful-boilerplate/infra/sqlite"
)

func main() {
	dbPath := flag.String("db", "./data.db", "path to sqlite database file")
	flag.Parse()

	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	ctx := context.Background()
	err = infradb.Migrate(ctx, db)
	_ = db.Close() //nolint:errcheck // best-effort close
	if err != nil {
		log.Fatalf("apply schema: %v", err)
	}
	log.Println("migration applied successfully")
}
