package cmd

var settings Settings

type Settings struct {
	DbPath         string
	ConfigDir      string
	ReloadDuration string
	Validity       string
	Email          string // for Let's Encrypt
}
