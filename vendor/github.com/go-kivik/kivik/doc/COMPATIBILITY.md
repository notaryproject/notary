# Key:

## Kivik components:

- ![Kivik API](images/api.png) : Supported by the Kivik Client API
- ![Kivik HTTP Server](images/http.png) : Supported by the Kivik HTTP Server
- ![Kivik Test Suite](images/tests.png) : Supported by the Kivik test suite
- ![CouchDB Logo](images/couchdb.png) : Supported by CouchDB backend
- ![PouchDB Logo](images/pouchdb.png) : Supported by PouchDB backend
- ![Memory Driver](images/memory.png) : Supported by Kivik Memory backend
- ![Filesystem Driver](images/filesystem.png) : Supported by the Kivik Filesystem backend

## API Functionality

- ✅ Yes : This feature is fully supported
- ☑️ Partial : This feature is partially supported
- ⍻ Emulated : The feature does not exist in the native driver, but is emulated.
- ？ Untested : This feature has been implemented, but is not yet fully tested.
- ⁿ/ₐ Not Applicable : This feature is supported, and doesn't make sense to emulate.
- ❌ No : This feature is supported by the backend, but there are no plans to add support to Kivik

<a name="authTable">

| Authentication Method | ![Kivik HTTP Server](images/http.png) | ![Kivik Test Suite](images/tests.png) | ![CouchDB](images/couchdb.png) | ![PouchDB](images/pouchdb.png) | ![Memory Driver](images/memory.png) | ![Filesystem Driver](images/filesystem.png) |
|--------------|:-------------------------------------:|:-------------------------------------:|:------------------------------:|:------------------------------:|:-----------------------------------:|:------------------------------------------:|
| HTTP Basic Auth    | ✅ | ✅ | ✅ | ✅<sup>[1](#pouchDbAuth)</sup> | ⁿ/ₐ | ⁿ/ₐ<sup>[2](#fsAuth)</sup>
| Cookie Auth        | ✅ | ✅ | ✅<sup>[3](#couchGopherJSAuth)</sup> |    | ⁿ/ₐ | ⁿ/ₐ<sup>[2](#fsAuth)</sup>
| Proxy Auth         |    |    |    |    | ⁿ/ₐ | ⁿ/ₐ<sup>[2](#fsAuth)</sup>

### Notes

1. <a name="pouchDbAuth"> PouchDB Auth support is only for remote databases. Local databases rely on a same-origin policy.
2. <a name="fsAuth">The Filesystem driver depends on whatever standard filesystem permissions are implemented by your operating system. This means that you do have the option on a Unix filesystem, for instance, to set read/write permissions on a user/group level, and Kivik will naturally honor these, and report access denied errors as one would expect.
3. <a name="couchGopherJSAuth">Due to security limitations in the XMLHttpRequest spec, when compiling the standard CouchDB driver with GopherJS, CookieAuth will not work.

| API Endpoint | ![Kivik API](images/api.png) | ![Kivik HTTP Server](images/http.png) | ![Kivik Test Suite](images/tests.png) | ![CouchDB](images/couchdb.png) | ![PouchDB](images/pouchdb.png) | ![Memory Driver](images/memory.png) | ![Filesystem Driver](images/filesystem.png) |
|---------------------------------------|----------------------|:-------------------------------------:|:-------------------------------------:|:------------------------------:|:------------------------------:|:-----------------------------------:|:------------------------------------------:|
| GET /                                 | ServerInfo()         | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| GET /_active_tasks                    | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_all_dbs                         | AllDBs()             | ✅ | ✅ | ✅ | ☑️<sup>[1](#pouchAllDbs1),[2](#pouchAllDbs2),[3](pouchLocalOnly)</sup> | ✅ | ✅
| GET /_db_updates                      | DBUpdates()          |    | ✅ | ✅ | ⁿ/ₐ |
| GET /_log                             | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_replicate                       | Replicate()          |    | ✅ | ✅<sup>[4](#replicator)</sup> | ✅ |
| GET /_restart                         | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_stats                           | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_utils                           | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_uuids                           | ⁿ/ₐ                   |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_membership                      | ⁿ/ₐ                   | ❌<sup>[12](#kivikCluster)</sup> |   | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ
| GET /favicon.ico                      | ⁿ/ₐ                  | ✅ | ❌ | ❌ | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| POST /_session<sup>[6](#cookieAuth)</sup> | ⁿ/ₐ<sup>[13](#getSession)</sup> | ✅ | ✅ | ✅ | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| GET /_session<sup>[6](#cookieAuth)</sup> | Session()        | ☑️ | ✅ | ✅ | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| DELETE /_session<sup>[6](#cookieAuth)</sup> | ⁿ/ₐ<sup>[13](#getSession)</sup> | ✅ | ✅ | ✅ | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| * /_config                            | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ | ⁿ/ₐ | ⁿ/ₐ |
| HEAD /{db}                            | DBExists()          | ✅ | ✅ | ✅ | ✅<sup>[5](#pouchDBExists)</sup> | ✅ | ✅
| GET /{db}                             | Stats()             | ✅ | ✅ | ✅ | ✅ |   | ☑️
| PUT /{db}                             | CreateDB()          | ✅ | ✅ | ✅ | ✅<sup>[5](#pouchDBExists)</sup> | ✅ | ✅
| DELETE /{db}                          | DestroyDB()         |    | ✅ | ✅ | ✅<sup>[5](#pouchDBExists)</sup> | ✅ | ✅
| POST /{db}                            | CreateDoc()         |    | ✅ | ✅ | ✅ | ✅ |
| (GET\|POST) /{db}/_all_docs           | AllDocs()           |    | ☑️<sup>[7](#todoConflicts),[9](#todoOrdering),[10](#todoLimit)</sup> | ✅ | ？ | ☑️<sup>[19](#memstatus)</sup> |
| POST /{db}/_bulk_docs                 | BulkDocs()          |    | ✅ | ✅ | ✅ | ⍻ |    |
| POST /{db}/_find                      | Find()              |    | ✅ | ✅ | ✅ |
| POST /{db}/_index                     | CreateIndex()       |    | ✅ | ✅ | ✅ |
| GET /{db}/_index                      | GetIndexes()        |    | ✅ | ✅ | ✅ |
| DELETE /{db}/_index                   | DeleteIndex()       |    | ✅ | ✅ | ✅ |
| POST /{db}/_explain                   | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> |    |
| (GET\|POST) /{db}/_changes            | Changes()<sup>[8](#changesContinuous)</sup> |    | ✅ | ✅ | ✅ |    |    |
| POST /{db}/_compact                   | Compact()           |    | ✅ | ✅ | ✅ |     |    |
| POST /{db}/_compact/{ddoc}            | CompactView()       |    |    | ✅ | ⁿ/ₐ |    |    |
| POST /{db}/_ensure_full_commit        | Flush()             | ✅ | ✅ | ✅ | ⁿ/ₐ | ⁿ/ₐ |    |
| POST /{db}/_view_cleanup              | ViewCleanup()       |    | ✅ | ✅ | ✅ |     |    |
| GET /{db}/_security                   | Security()          |    | ✅ | ✅ | ⁿ/ₐ<sup>[14](#pouchPlugin)</sup> | ✅
| PUT /{db}/_security                   | SetSecurity()       |    | ✅ | ✅ | ⁿ/ₐ<sup>[14](#pouchPlugin)</sup> | ✅
| POST /{db}/_temp_view                 | ⁿ/ₐ                  | ⁿ/ₐ | ⁿ/ₐ| ⁿ/ₐ<sup>[16](#tempViews)</sup> | ⁿ/ₐ<sup>[17](#pouchTempViews)</sup> | ⁿ/ₐ | ⁿ/ₐ |
| POST /{db}/_purge                     | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_missing_revs              | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_revs_diff                 | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| GET /{db}/_revs_limit                 | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| PUT /{db}/_revs_limit                 | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| HEAD /{db}/{docid}                    | Rev()               |    | ✅ | ✅ | ⍻ | ⍻
| GET /{db}/{docid}                     | Get()               |    | ☑️<sup>[7](#todoConflicts),[11](#todoAttachments)</sup> | ✅ | ✅ | ☑️<sup>[18](#memstatus)</sup>
| PUT /{db}/{docid}                     | Put()               |    | ☑️<sup>[11](#todoAttachments)</sup> | ✅ | ✅ | ☑️<sup>[18](#memstatus)</sup>
| DELETE /{db}/{docid}                  | Delete()            |    | ✅ | ✅ | ✅ | ✅
| COPY /{db}/{docid}                    | Copy()              |    | ✅ | ✅ | ⍻ |
| HEAD /{db}/{docid}/{attname}          | GetAttachmentMeta() |    | ✅ | ✅ | ⍻ |
| GET /{db}/{docid}/{attname}           | GetAttachment()     |    | ✅ | ✅ | ✅ |
| PUT /{db}/{docid}/{attname}           | PutAttachment()     |    | ✅ | ✅ | ✅ |
| DELETE /{db}/{docid}/{attname}        | DeleteAttachment()  |    | ✅ | ✅ | ✅ |
| HEAD /{db}/_design/{ddoc}             | Rev()               |    | ✅ | ✅ | ✅ |
| GET /{db}/_design/{ddoc}              | Get()               |    | ✅ | ✅ | ✅ |
| PUT /{db}/_design/{ddoc}              | Put()               |    | ✅ | ✅ | ✅ |
| DELETE /{db}/_design/{ddoc}           | Delete()            |    | ✅ | ✅ | ✅ |
| COPY /{db}/_design/{ddoc}             | Copy()              |    | ✅ | ✅ | ⍻ |
| HEAD /{db}/_design/{ddoc}/{attname}   | GetAttachmentMeta() |    | ✅ | ✅ | ✅ |
| GET /{db}/_design/{ddoc}/{attname}    | GetAttachment()     |    | ✅ | ✅ | ✅ |
| PUT /{db}/_design/{ddoc}/{attname}    | PutAttachment()     |    | ✅ | ✅ | ✅ |
| DELETE /{db}/_design/{ddoc}/{attname} | DeleteAttachment()  |    | ✅ | ✅ | ✅ |
| GET /{db}/_design/{ddoc}/_info        | ⁿ/ₐ                  |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| (GET\|POST) /{db}/_design/{ddoc}/_view/{view} | Query()     |    | ✅ | ✅ | ✅<sup>[18](#pouchViews)</sup> |
| GET /{db}/_design/{ddoc}/_show/{func} | ⁿ/ₐ |    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_design/{ddoc}/_show/{func} | ⁿ/ₐ|    |    | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| GET /{db}/_design/{ddoc}/_show/{func}/{docid} |ⁿ/ₐ| | | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_design/{ddoc}/_show/{func}/{docid} |ⁿ/ₐ| | |❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| GET /{db}/_design/{ddoc}/_list/{func}/{view} | ⁿ/ₐ| | | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_design/{ddoc}/_list/{func}/{view} |ⁿ/ₐ| | | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| GET /{db}/_design/{ddoc}/_list/{func}/{other-ddoc}/{view} |ⁿ/ₐ| | |❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_design/{ddoc}/_list/{func}/{other-ddoc}/{view} |ⁿ/ₐ| | |❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| POST /{db}/_design/{ddoc}/_update/{func} | ⁿ/ₐ |   |   |❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| PUT /{db}/_design/{ddoc}/_update/{func}/{docid} |ⁿ/ₐ| | |❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| ANY /{db}/_design/{ddoc}/_rewrite/{path} | ⁿ/ₐ |  |   | ❌<sup>[15](#notPublic)</sup> | ⁿ/ₐ |
| HEAD /{db}/_local/{docid}   | Rev()               |    | ✅ | ✅ | ✅ |
| GET /{db}/_local/{docid}    | Get()               |    | ✅ | ✅ | ✅ |
| PUT /{db}/_local/{docid}    | Put()               |    | ✅ | ✅ | ✅ |
| DELETE /{db}/_local/{docid} | Delete()            |    | ✅ | ✅ | ✅ |
| COPY /{db}/_local/{docid}   | Copy()              |    | ✅ | ✅ | ⍻ |

### Notes

1. <a name="pouchAllDbs1"> PouchDB support for AllDbs depends on the
    [pouchdb-all-dbs plugin](https://github.com/nolanlawson/pouchdb-all-dbs).
2. <a name="pouchAllDbs2"> Unit tests broken in PouchDB due to an [apparent
    bug](https://github.com/nolanlawson/pouchdb-all-dbs/issues/25) in the
    pouchdb-all-dbs plugin.
3. <a name="pouchLocalOnly"> Supported for local PouchDB databases only. A work
    around may be possible in the future for remote databases.
4. <a name="replicator"> Replications are actually done via the _replicator
   database, not the /_replicate endpoint.
5. <a name="pouchDBExists"> PouchDB offers no way to check for the existence of
   a local database without creating it, so `DBExists()` always returns true,
   `CreateDB()` does not return an error if the database already existed, and
   `DestroyDB()` does not return an error if the database does not exist.
6. <a name="cookieAuth"> See the CookieAuth section in the [Authentication methods table](#authTable)
7. <a name="todoConflicts"> **TODO:** Conflicts are not yet tested.
8. <a name="changesContinuous"> Changes feed operates in continuous mode only.
9. <a name="todoOrdering"> **TODO:** Ordering is not yet tested.
10. <a name="todoLimit"> **TODO:** Limits are not yet tested.
11. <a name="todoAttachments"> **TODO:** Attachments are not yet tested.
12. <a name="kivikCluster"> There are no plans at present to support clustering.
13. <a name="getSession"> Used for authentication, but not exposed directly to
    the client API.
14. <a name="pouchPlugin"> This feature is not available in the core PouchDB
    package. Support is provided in PouchDB plugins, so including optional
    support here may be possiblein the future.
15. <a name="notPublic"> This feature is not considered (by me, if nobody else)
    part of the public CouchDB API, or otherwise not meaningful to make part of
    the Kivik API, so there are no (immediate) plans to implement support. If
    you feel this should change for a given feature, please create an issue and
    explain your reasons.
16. <a name="tempViews"> As of CouchDB 2.0, temp views are no longer supported,
    so I see no reason to support them in this library for older server versions.
    If you feel they should be supported, please create an issue and make your
    case.
17. <a name="pouchTempViews"> At present, PouchDB effectively supports temp
    views by calling [query](https://pouchdb.com/api.html#query_database) with
    a JS function. This feature is scheduled for removal from PouchDB (into a
    plugin), but until then, this functionality can still be used via the
    Query() method, by passing a JS function as an option.
18. <a name="pouchViews"> Only queries against defined design documents are
    supported. That is to say, providing raw JS functions is not supported. If
    you need this, please create an issue to make your case.
19. <a name="memstatus"> See [Issue #142](https://github.com/go-kivik/kivik/issues/142)
    for the current status of the memory driver.

## HTTP Status Codes

The CouchDB API prescribes some status codes which, to me, don't make a lot of
sense. This is particularly true of a few error status codes. It seems the folks
at [Cloudant](https://cloudant.com/) share my opinion, as they have changed some
as well.

In particular, the CouchDB API returns a status 500 **Internal Server Error** for
quite a number of malformed requests.  Example: `/_uuids?count=-1` will return
500.  Cloudant and Kivik both return 400 **Bad Request** in this case, and in
many other cases as well, as this seems to better reflect the actual state.
