package main

import (
	"database/sql"
	"log"

	_ "github.com/marcboeker/go-duckdb/v2"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	stmt := `COPY (SELECT * FROM read_csv_auto('testdata/tiny.csv', header=true))
	         TO 'testdata/tiny.parquet' (FORMAT parquet)`
	if _, err := db.Exec(stmt); err != nil {
		log.Fatal(err)
	}
	log.Println("wrote testdata/tiny.parquet")
}
