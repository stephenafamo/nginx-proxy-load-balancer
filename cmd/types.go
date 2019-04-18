package cmd

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
	SslSource       string // letsencrypt, proxy, manual
	HttpsOnly       bool
	CertPath        string
	KeyPath         string

	// parameters for TCP/UDP proxy type
	Port          uint // required for this type
	ServerOptions Options
}

type ConfigTemplateStruct struct {
	ServiceConfig
	Unique   string
}
