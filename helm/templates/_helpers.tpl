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

{{- define "notary.serverdbhostname" -}}
{{- if .Values.storage.host -}}
{{ .Values.storage.host }}
{{- else -}}
{{ include "notary.fullname" . }}-db
{{- end -}}
{{- end -}}


{{- define "notary.signerdbhostname" -}}
{{- if .Values.storage.host -}}
{{ .Values.storage.host }}
{{- else -}}
{{ include "notary.fullname" . }}-db
{{- end -}}
{{- end -}}

{{- define "notary.serverdburl" -}}
{{- if .Values.storage.dbUrl -}}
{{ .Values.storage.dbUrl }}
{{- else -}}
{{- if eq .Values.storage.type "mysql" -}}
root@tcp({{ template "notary.serverdbhostname" . }}:3306)/notaryserver
{{- else if eq .Values.storage.type "postgres" -}}
server@{{ template "notary.serverdbhostname" . }}:5432/notaryserver?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-server.pem&sslkey=/tls/notary-server-key.pem
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "notary.signerdburl" -}}
{{- if .Values.storage.dbUrl -}}
{{ .Values.storage.dbUrl }}
{{- else -}}
{{- if eq .Values.storage.type "mysql" -}}
root@tcp({{ template "notary.signerdbhostname" . }}:3306)/notarysigner
{{- else if eq .Values.storage.type "postgres" -}}
signer@{{ template "notary.signerdbhostname" . }}:5432/notarysigner?sslmode=verify-ca&sslrootcert=/tls/database-ca.pem&sslcert=/tls/notary-signer.pem&sslkey=/tls/notary-signer-key.pem"
{{- end -}}
{{- end -}}
{{- end -}}