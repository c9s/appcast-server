package main

import (
	"database/sql"
	"log"
	"os"
)

func ConnectDB(dbname string) *sql.DB {
	var initDB = false
	_, err := os.Stat(dbname)
	if os.IsNotExist(err) {
		// init db
		initDB = true
	}

	// os.Remove("./appcast.db")
	db, err := sql.Open("sqlite3", dbname)
	if err != nil {
		log.Fatal(err)
	}

	if initDB {
		log.Println("Initializing database schema...")
		prepareTables(db)
	}
	return db
}

func prepareTables(db *sql.DB) {
	createAccountTable(db)
	createReleaseTable(db)
	createChannelTable(db)
	createDownloadLogTable(db)
}

func createAccountTable(db *sql.DB) {
	if _, err := db.Exec(`
	CREATE TABLE accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account varchar,
		token varchar
	);`); err != nil {
		log.Fatal(err)
	}
}

func createChannelTable(db *sql.DB) {
	if _, err := db.Exec(`CREATE TABLE channels(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title varchar,
		description text,
		identity varchar,
		token varchar
	);`); err != nil {
		log.Fatal(err)
	}
	/*
		http://localhost:8080/appcast/gotray/4cbd040533a2f43fc6691d773d510cda70f4126a
	*/
	if _, err := db.Exec(`INSERT INTO channels(title,description, identity, token) values (?,?,?,?)`, "GoTray", "Desc", "gotray", "4cbd040533a2f43fc6691d773d510cda70f4126a"); err != nil {
		panic(err)
	}
}

func createDownloadLogTable(db *sql.DB) {
	if _, err := db.Exec(`
	CREATE TABLE download_logs(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		remote_addr varchar,
		request_uri varchar,
		referer varchar,
		user_agent varchar,
		release_id integer
	);
	`); err != nil {
		log.Fatal(err)
	}
}

func createReleaseTable(db *sql.DB) {
	if _, err := db.Exec(`CREATE TABLE releases(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title varchar,
		desc text,
		release_notes text,
		pubdate datetime default current_timestamp,
		filename varchar,
		channel varchar,
		length integer,
		mimetype varchar,
		version varchar,
		short_version_string varchar,
		dsa_signature varchar,
		token varchar,
		downloaded integer default 0
	);`); err != nil {
		log.Fatal(err)
	}
}
