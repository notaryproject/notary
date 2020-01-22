package client

// NotaryConfig notary configuration
type NotaryConfig struct {
	TrustDir     string             `json:"trust_dir"`
	RemoteServer RemoteServerConfig `json:"remote_server"`
	TrustPinning TrustPinningConfig `json:"trust_pinning"`
}

// RemoteServerConfig notary remote server configuration
type RemoteServerConfig struct {
	URL           string `json:"url"`
	RootCA        string `json:"root_ca"`
	TLSClientKey  string `json:"tls_client_key"`
	TLSClientCert string `json:"tls_client_cert"`
	SkipTLSVerify bool   `json:"skipTLSVerify"`
}

// TrustPinningConfig notary trust pinning configuration
type TrustPinningConfig struct {
	DisableTofu bool                   `json:"disable_tofu"`
	CA          map[string]string      `json:"ca"`
	Certs       map[string]interface{} `json:"certs"`
}
