FROM couchdb:2.1.1

COPY fixtures/couchdb /opt/couchdb.conf/

COPY couchdb/local.ini /opt/couchdb/etc/local.ini

COPY couchdb/couchdb_startup.sh /

ENTRYPOINT ["/usr/bin/env", "bash"]
CMD ["-c", "bash /couchdb_startup.sh"]
