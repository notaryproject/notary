{
	"server": {
		"grpc_addr": ":{{ .Values.signer.port }}",
		"tls_cert_file": "/tls/notary-signer.crt",
		"tls_key_file": "/tls/notary-signer.key",
		"client_ca_file": "/tls/root-ca.crt"
	},
	"logging": {
		"level": "{{ .Values.logging.level }}"
	},
	"storage": {
		"backend": "{{ .Values.storage.type }}",
		"db_url": "{{ template "notary.signerdburl" . }}",
		"default_alias": "{{ .Values.signer.defaultAlias }}"
	}
}