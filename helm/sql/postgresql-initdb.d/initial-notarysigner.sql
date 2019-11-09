CREATE DATABASE notarysigner;
CREATE USER {{ .Values.signer.storageCredentials.user }};
GRANT ALL PRIVILEGES ON DATABASE notarysigner TO {{ .Values.signer.storageCredentials.user }};
