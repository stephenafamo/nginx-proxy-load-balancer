package main

import (
	"github.com/volatiletech/sqlboiler-sqlite3/driver"
	"github.com/volatiletech/sqlboiler/v4/drivers"
)

func main() {
	drivers.DriverMain(&driver.SQLiteDriver{})
}
