CREATE DATABASE IF NOT EXISTS `notaryserver`;

CREATE USER "{{ .Values.server.storageCredentials.user }}"@"%" IDENTIFIED BY "{{ .Values.server.storageCredentials.password }}";

GRANT
	ALL PRIVILEGES ON `notaryserver`.* 
	TO "{{ .Values.server.storageCredentials.user }}"@"%";
