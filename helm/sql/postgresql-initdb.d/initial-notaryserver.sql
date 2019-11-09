CREATE DATABASE notaryserver;
CREATE USER {{ .Values.server.storageCredentials.user }};
GRANT ALL PRIVILEGES ON DATABASE notaryserver TO {{ .Values.server.storageCredentials.user }};
