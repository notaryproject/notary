package client

// NotaryConfig notary configuration
type NotaryConfig struct {
	TrustDir     string             `json:"trust_dir" mapstructure:"trust_dir"`
	RemoteServer RemoteServerConfig `json:"remote_server" mapstructure:"remote_server"`
	TrustPinning TrustPinningConfig `json:"trust_pinning" mapstructure:"trust_pinning"`
}

// RemoteServerConfig notary remote server configuration
type RemoteServerConfig struct {
	URL           string `json:"url" mapstructure:"url"`
	RootCA        string `json:"root_ca" mapstructure:"root_ca"`
	TLSClientKey  string `json:"tls_client_key" mapstructure:"tls_client_key"`
	TLSClientCert string `json:"tls_client_cert" mapstructure:"tls_client_cert"`
	SkipTLSVerify bool   `json:"skipTLSVerify" mapstructure:"skipTLSVerify"`
}

// TrustPinningConfig notary trust pinning configuration
type TrustPinningConfig struct {
	DisableTofu bool                   `json:"disable_tofu" mapstructure:"disable_tofu"`
	CA          map[string]string      `json:"ca" mapstructure:"ca"`
	Certs       map[string]interface{} `json:"certs" mapstructure:"certs"`
}
