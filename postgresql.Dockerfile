FROM postgres:9.5.4
COPY ./notarysql/postgresql-initdb.d /docker-entrypoint-initdb.d
