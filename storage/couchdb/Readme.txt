The goal of the CouchDB driver is to integrate the notary signer and
server with CouchDB as well as the cloud version of CouchDB, which is
called 'Cloudant'. The former is available as a couchdb container,
the latter through the IBM Cloud.

A good introduction for the CouchDB can be found here:

  http://guide.couchdb.org/draft/tour.html


The support for the CouchDB and Cloudant requires us to accommodate
to the requirements and limitation of their configurations. Both may
be based on the same version of CouchDB, which is currently 2.1.1.

The main difference between CouchDB and Cloudant is the client usage
model. For a CouchDB the administrator can specify the admin username
and password for example when starting a CouchDB container. The admin
account allows to create other user accounts and give privileges to
them for accessing databases for example. A Cloudant may be set up in
a more restrictive way and its primary user may not be able to create
other user accounts.
