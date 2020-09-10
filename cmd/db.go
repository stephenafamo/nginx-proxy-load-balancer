package cmd

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const (
	stateNotConfigured    = "not configured"
	stateToConfigureHttps = "to configure https"
	stateToDisableHttp    = "to disable http"
	stateConfigured       = "configured"
)

func createTables(db *sql.DB) error {

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS files (
		id INTEGER NOT NULL PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		is_configured BOOLEAN NOT NULL DEFAULT FALSE,
		last_modified DATETIME NOT NULL
	);`)
	if err != nil {
		return err
	}

	// name is the name of the service in the config file
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS services (
		id INTEGER NOT NULL PRIMARY KEY,
		file_id INTEGER REFERENCES files (id) ON DELETE CASCADE ON UPDATE CASCADE,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		state TEXT NOT NULL,
		is_ssl BOOLEAN NOT NULL,
		https_configured DATETIME,
		last_modified DATETIME NOT NULL
	);`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS nginx_configs (
		id INTEGER NOT NULL PRIMARY KEY,
		service_id INTEGER REFERENCES services (id) ON DELETE SET NULL ON UPDATE CASCADE,
		type TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		last_modified DATETIME NOT NULL
	);`)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
