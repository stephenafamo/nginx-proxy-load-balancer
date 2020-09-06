package internal

import "time"

type Settings struct {
	TESTING bool `env:"TESTING"`

	EMAIL string `env:"EMAIL,required"` // for Let's Encrypt

	DB_PATH            string        `env:"DB_PATH,default=./db"`
	CONFIG_DIR         string        `env:"CONFIG_DIR,default=./config"`
	CONFIG_RELOAD_TIME time.Duration `env:"CONFIG_RELOAD_TIME,default=5s"`
	HTTPS_VALIDITY     time.Duration `env:"HTTPS_VALIDITY,default=168h"` // 7 days

	CONFIG_OUTPUT_DIR           string `env:"CONFIG_OUTPUT_DIR,default=/etc/nginx/conf.d"`
	LETSENCRYPT_CREDS_DIR       string `env:"LETSENCRYPT_CREDS_DIR,default=./letsencrypt-credentials"`
	LETSENCRYPT_DNS_PROPAGATION int    `env:"LETSENCRYPT_DNS_PROPAGATION,default=120"`

	SENTRY_DSN string `env:"SENTRY_DSN"`
}

type Options map[string]string

type Location struct {
	Match           string
	Options         Options
	Upstream        []UpstreamServer
	UpstreamOptions Options
}

type UpstreamServer struct {
	Address    string
	Parameters []string
}

type ServiceConfig struct {
	Type            string // HTTP, TCP, SNI, default HTTP
	Upstream        []UpstreamServer
	UpstreamOptions Options

	// Parameters for HTTP proxy type
	Domains         []string // required for this type
	Location        string   // Default "/"
	LocationOptions Options
	Locations       []Location
	Ssl             bool
	SslSource       string // manual, proxy, letsencrypt, letsencrypt-manual
	HttpsOnly       bool
	CertPath        string
	KeyPath         string

	// If this is provided, the appropriate letsencrypt dns plugin is used
	// NOTE: if using --dns-digitalocean, this should be "digitalocean" only
	//
	// Aside from the plugins from certbot, some other plugins have been implemented
	// in this program (e.g. vultr)
	LetsEncryptDNSPlugin string

	// If these are provided, the manual certbot plugin will be used
	// instead of the webroot or dns plugin which is automatic completed
	// These should be passed to executables which will be the "hooks"
	LetsEncryptAuthenticator string
	LetsEncryptCleaner       string

	// parameters for TCP/UDP proxy type
	Port          uint // required for this type
	ServerOptions Options
}

type Config struct {
	ServiceConfig
	Unique string
}
