{{- define "notary.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "notary.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "notary.serverdburl" -}}
{{- if eq .Values.storage.flavor "mysql" -}}
{{- if .Values.storage.remote.enabled -}}
root@tcp({{ .Values.storage.remote.host }}:{{ .Values.storage.remote.port }})/notaryserver
{{- else -}}
{{ .Values.server.storageCredentials.user }}:%% .Env.PASSWORD %%@tcp(notary-db:3306)/notaryserver
{{- end }}
{{- else if eq .Values.storage.flavor "postgres" -}}
{{- if .Values.storage.remote.enabled -}}
server@{{ .Values.storage.remote.host }}:{{ .Values.storage.remote.port }}/notaryserver?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-server.pem&sslkey=/tls/notary-server-key.pem"
{{- else -}}
server@notary-db:5432/notaryserver?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-server.pem&sslkey=/tls/notary-server-key.pem"
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "notary.signerdburl" -}}
{{- if eq .Values.storage.flavor "mysql" -}}
{{- if .Values.storage.remote.enabled -}}
root@tcp({{ .Values.storage.remote.host }}:{{ .Values.storage.remote.port }})/notarysigner
{{- else -}}
{{ .Values.signer.storageCredentials.user }}:%% .Env.PASSWORD %%@tcp(notary-db:3306)/notarysigner
{{- end }}
{{- else if eq .Values.storage.flavor "postgres" -}}
{{- if .Values.storage.remote.enabled -}}
signer@{{ .Values.storage.remote.host }}:{{ .Values.storage.remote.port }}/notarysigner?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-signer.pem&sslkey=/tls/notary-signer-key.pem"
{{- else -}}
signer@notary-db:5432/notarysigner?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-signer.pem&sslkey=/tls/notary-signer-key.pem"
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "notary.gunprefixes" -}}
{{- .Values.server.gunPrefixes | toJson -}}
{{ end -}}