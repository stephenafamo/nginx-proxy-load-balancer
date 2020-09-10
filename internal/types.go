package internal

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"

	"github.com/BurntSushi/toml"
)

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

type Config struct {
	Service
	Unique string
}

type ServiceMap map[string]Service

type Service struct {
	Type            string // HTTP, TCP default HTTP
	Upstream        []UpstreamServer
	UpstreamOptions Options

	// Parameters for HTTP proxy type
	// Required for this type. Domains to proxy
	Domains []string

	// Default "/". will be used as "match" for default "Locations"
	Location        string
	LocationOptions Options
	Locations       []Location

	Ssl       bool   // Whether to generate HTTPS configutation
	HttpsOnly bool   // Wether to automatically redirect http to https. Default false
	SslSource string // Required if Ssl = true. Options: manual, letsencrypt
	CertPath  string // If using manual sslSource
	KeyPath   string // If using manual sslSource
	// If this is provided, the appropriate letsencrypt dns plugin is used
	// NOTE: if using --dns-digitalocean, this should be "digitalocean" only
	//
	// Aside from the plugins from certbot, some other plugins have been implemented
	// in this program (e.g. vultr)
	LetsEncryptDNSPlugin string
	// If these are provided, the manual certbot plugin will be used
	// instead of the webroot or dns plugin which is automatic completed
	// These should be paths to executables which will be the "hooks"
	// See https://certbot.eff.org/docs/using.html#pre-and-post-validation-hooks
	LetsEncryptAuthenticator string
	LetsEncryptCleaner       string

	// parameters for TCP/UDP proxy type
	Port          uint    // REQUIRED for this type
	ServerOptions Options // Optional

	// A grpc endpoint to send notifications about the configuration stauts
	Webhook *Webhook
}

type Location struct {
	// REQUIRED: the path of the request to proxy. See
	Match string
	// Optional: other options to be added in the location block
	// See http://nginx.org/en/docs/http/ngx_http_core_module.html#location
	Options Options

	Upstream []UpstreamServer
	// Optional: extra directives to the upstream block for fine tuning. See http://nginx.org/en/docs/http/ngx_http_upstream_module.html
	UpstreamOptions Options
}

type UpstreamServer struct {
	// REQUIRED: the address of the upstream server
	// Must be reachable
	Address string

	// Optional: other parameters for the upstream. See http://nginx.org/en/docs/http/ngx_http_upstream_module.html#server
	Parameters []string
}

type Options = map[string]string

type Webhook struct {
	URL url.URL
}

func (w *Webhook) UnmarshalText(text []byte) error {
	textStr := string(text)
	theURL, err := url.Parse(textStr)
	if err != nil {
		return fmt.Errorf("could not unmarshal %q into url: %w", textStr, err)
	}

	w.URL = *theURL
	return nil
}

// Value implements the driver Valuer interface.
func (u ServiceMap) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(u)
	return buf.Bytes(), err
}

// Scan implements the Scanner interface.
func (u *ServiceMap) Scan(value interface{}) error {
	var err error

	switch x := value.(type) {
	case string:
		_, err = toml.Decode(x, u)
	case []byte:
		_, err = toml.Decode(string(x), u)
	case nil:
		return nil

	default:
		err = fmt.Errorf("cannot scan type %T into type Service: %v", value, value)
	}

	return err
}

// Value implements the driver Valuer interface.
func (u Service) Value() (driver.Value, error) {
	buf := &bytes.Buffer{}
	err := toml.NewEncoder(buf).Encode(u)
	return buf.Bytes(), err
}

// Scan implements the Scanner interface.
func (u *Service) Scan(value interface{}) error {
	var err error

	switch x := value.(type) {
	case string:
		_, err = toml.Decode(x, u)
	case []byte:
		_, err = toml.Decode(string(x), u)
	case nil:
		return nil

	default:
		err = fmt.Errorf("cannot scan type %T into type Service: %v", value, value)
	}

	return err
}
