package cmd

var settings Settings

type Settings struct {
	DbPath         string
	ConfigDir      string
	ReloadDuration string
	PurgeDuration string
	Validity       string
	Email          string // for Let's Encrypt
}
