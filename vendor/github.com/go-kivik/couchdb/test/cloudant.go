package test

import (
	"github.com/go-kivik/kivik"
	"github.com/go-kivik/kiviktest"
	"github.com/go-kivik/kiviktest/kt"
)

func registerSuiteCloudant() {
	kiviktest.RegisterSuite(kiviktest.SuiteCloudant, kt.SuiteConfig{
		"AllDBs.expected":               []string{"_replicator", "_users"},
		"AllDBs/NoAuth.status":          kivik.StatusUnauthorized,
		"AllDBs/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"CreateDB/RW/NoAuth.status":         kivik.StatusUnauthorized,
		"CreateDB/RW/Admin/Recreate.status": kivik.StatusPreconditionFailed,

		"DestroyDB/RW/Admin/NonExistantDB.status":  kivik.StatusNotFound,
		"DestroyDB/RW/NoAuth/NonExistantDB.status": kivik.StatusNotFound,
		"DestroyDB/RW/NoAuth/ExistingDB.status":    kivik.StatusUnauthorized,

		"AllDocs.databases":                  []string{"_replicator", "chicken", "_duck"},
		"AllDocs/Admin/_replicator.expected": []string{"_design/_replicator"},
		"AllDocs/Admin/_replicator.offset":   0,
		"AllDocs/Admin/chicken.status":       kivik.StatusNotFound,
		"AllDocs/Admin/_duck.status":         kivik.StatusForbidden,
		"AllDocs/NoAuth/_replicator.status":  kivik.StatusUnauthorized,
		"AllDocs/NoAuth/chicken.status":      kivik.StatusNotFound,
		"AllDocs/NoAuth/_duck.status":        kivik.StatusUnauthorized,
		"AllDocs/RW/group/NoAuth.status":     kivik.StatusUnauthorized,

		"Find.databases":                         []string{"_replicator", "chicken", "_duck"},
		"Find/Admin/_replicator.expected":        []string{"_design/_replicator"},
		"Find/Admin/_replicator.offset":          0,
		"Find/Admin/chicken.status":              kivik.StatusNotFound,
		"Find/Admin/_duck.status":                kivik.StatusForbidden,
		"Find/NoAuth/_replicator.status":         kivik.StatusUnauthorized,
		"Find/NoAuth/chicken.status":             kivik.StatusNotFound,
		"Find/NoAuth/_duck.status":               kivik.StatusUnauthorized,
		"Find/RW/group/NoAuth.status":            kivik.StatusUnauthorized,
		"Find/Admin/_replicator/Warning.warning": "no matching index found, create an index to optimize query time",
		"Find/RW/group/Admin/Warning.warning":    "no matching index found, create an index to optimize query time",
		"Find/RW/group/NoAuth/Warning.warning":   "no matching index found, create an index to optimize query time",

		"Explain.databases":             []string{"chicken", "_duck"},
		"Explain/Admin/chicken.status":  kivik.StatusNotFound,
		"Explain/Admin/_duck.status":    kivik.StatusForbidden,
		"Explain/NoAuth/chicken.status": kivik.StatusNotFound,
		"Explain/NoAuth/_duck.status":   kivik.StatusUnauthorized,

		"Query/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"DBExists.databases":              []string{"_users", "chicken", "_duck"},
		"DBExists/Admin/_users.exists":    true,
		"DBExists/Admin/chicken.exists":   false,
		"DBExists/Admin/_duck.status":     kivik.StatusForbidden,
		"DBExists/NoAuth/_users.status":   kivik.StatusUnauthorized,
		"DBExists/NoAuth/_duck.status":    kivik.StatusUnauthorized,
		"DBExists/NoAuth/chicken.exists":  false,
		"DBExists/RW/group/Admin.exists":  true,
		"DBExists/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"Log/Admin.status":              kivik.StatusForbidden,
		"Log/NoAuth.status":             kivik.StatusUnauthorized,
		"Log/Admin/Offset-1000.status":  kivik.StatusBadRequest,
		"Log/NoAuth/Offset-1000.status": kivik.StatusBadRequest,

		"Version.version":        `^2\.0\.0$`,
		"Version.vendor":         `^IBM Cloudant$`,
		"Version.vendor_version": `^\d\d\d\d$`,

		"Get/RW/group/Admin/bogus.status":        kivik.StatusNotFound,
		"Get/RW/group/NoAuth/bob.status":         kivik.StatusUnauthorized,
		"Get/RW/group/NoAuth/bogus.status":       kivik.StatusUnauthorized,
		"Get/RW/group/NoAuth/_design/foo.status": kivik.StatusUnauthorized,
		"Get/RW/group/NoAuth/_local/foo.status":  kivik.StatusUnauthorized,

		"GetMeta/RW/group/Admin/bogus.status":        kivik.StatusNotFound,
		"GetMeta/RW/group/NoAuth/bob.status":         kivik.StatusUnauthorized,
		"GetMeta/RW/group/NoAuth/bogus.status":       kivik.StatusUnauthorized,
		"GetMeta/RW/group/NoAuth/_design/foo.status": kivik.StatusUnauthorized,
		"GetMeta/RW/group/NoAuth/_local/foo.status":  kivik.StatusUnauthorized,

		"Put/RW/NoAuth/Create.status": kivik.StatusUnauthorized,

		"Flush.databases":                     []string{"_users", "chicken", "_duck"},
		"Flush/Admin/chicken/DoFlush.status":  kivik.StatusNotFound,
		"Flush/Admin/_duck/DoFlush.status":    kivik.StatusForbidden,
		"Flush/NoAuth/chicken/DoFlush.status": kivik.StatusNotFound,
		"Flush/NoAuth/_users/DoFlush.status":  kivik.StatusUnauthorized,
		"Flush/NoAuth/_duck/DoFlush.status":   kivik.StatusUnauthorized,

		"Delete/RW/Admin/group/MissingDoc.status":       kivik.StatusNotFound,
		"Delete/RW/Admin/group/InvalidRevFormat.status": kivik.StatusBadRequest,
		"Delete/RW/Admin/group/WrongRev.status":         kivik.StatusConflict,
		"Delete/RW/NoAuth.status":                       kivik.StatusUnauthorized,

		"Session/Get/Admin.info.authentication_handlers":  "delegated,cookie,default,local",
		"Session/Get/Admin.info.authentication_db":        "_users",
		"Session/Get/Admin.info.authenticated":            "cookie",
		"Session/Get/Admin.userCtx.roles":                 "_admin,_reader,_writer",
		"Session/Get/Admin.ok":                            "true",
		"Session/Get/NoAuth.info.authentication_handlers": "delegated,cookie,default,local",
		"Session/Get/NoAuth.info.authentication_db":       "_users",
		"Session/Get/NoAuth.info.authenticated":           "local",
		"Session/Get/NoAuth.userCtx.roles":                "",
		"Session/Get/NoAuth.ok":                           "true",

		"Session/Post/EmptyJSON.status":                               kivik.StatusBadRequest,
		"Session/Post/BogusTypeJSON.status":                           kivik.StatusBadRequest,
		"Session/Post/BogusTypeForm.status":                           kivik.StatusBadRequest,
		"Session/Post/EmptyForm.status":                               kivik.StatusBadRequest,
		"Session/Post/BadJSON.status":                                 kivik.StatusBadRequest,
		"Session/Post/BadForm.status":                                 kivik.StatusBadRequest,
		"Session/Post/MeaninglessJSON.status":                         kivik.StatusInternalServerError,
		"Session/Post/MeaninglessForm.status":                         kivik.StatusBadRequest,
		"Session/Post/GoodJSON.status":                                kivik.StatusUnauthorized,
		"Session/Post/BadQueryParam.status":                           kivik.StatusUnauthorized,
		"Session/Post/BadCredsJSON.status":                            kivik.StatusUnauthorized,
		"Session/Post/BadCredsForm.status":                            kivik.StatusUnauthorized,
		"Session/Post/GoodCredsJSONRemoteRedirHeaderInjection.status": kivik.StatusBadRequest,
		"Session/Post/GoodCredsJSONRemoteRedirInvalidURL.skip":        true, // Cloudant doesn't sanitize the Location value, so sends unparseable headers.

		"Stats.databases":             []string{"_users", "chicken", "_duck"},
		"Stats/Admin/chicken.status":  kivik.StatusNotFound,
		"Stats/Admin/_duck.status":    kivik.StatusForbidden,
		"Stats/NoAuth/_users.status":  kivik.StatusUnauthorized,
		"Stats/NoAuth/chicken.status": kivik.StatusNotFound,
		"Stats/NoAuth/_duck.status":   kivik.StatusUnauthorized,
		"Stats/RW/NoAuth.status":      kivik.StatusUnauthorized,

		"CreateDoc/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"Compact/RW/Admin.status":  kivik.StatusForbidden,
		"Compact/RW/NoAuth.status": kivik.StatusUnauthorized,

		"ViewCleanup/RW/Admin.status":  kivik.StatusForbidden,
		"ViewCleanup/RW/NoAuth.status": kivik.StatusUnauthorized,

		"Security.databases":                    []string{"_replicator", "_users", "_global_changes", "chicken", "_duck"},
		"Security/Admin/_global_changes.status": kivik.StatusForbidden,
		"Security/Admin/chicken.status":         kivik.StatusNotFound,
		"Security/Admin/_duck.status":           kivik.StatusForbidden,
		"Security/NoAuth.status":                kivik.StatusUnauthorized,
		"Security/NoAuth/chicken.status":        kivik.StatusNotFound,
		"Security/NoAuth/_duck.status":          kivik.StatusUnauthorized,
		"Security/RW/group/NoAuth.status":       kivik.StatusUnauthorized,

		"SetSecurity/RW/Admin/NotExists.status":  kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/NotExists.status": kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/Exists.status":    kivik.StatusUnauthorized,

		"DBUpdates/RW/Admin.status":  kivik.StatusNotFound, // Cloudant apparently disables this
		"DBUpdates/RW/NoAuth.status": kivik.StatusUnauthorized,

		"Changes/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"Copy/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"BulkDocs/RW/NoAuth.status":                    kivik.StatusUnauthorized,
		"BulkDocs/RW/NoAuth/group/Mix/Conflict.status": kivik.StatusConflict,
		"BulkDocs/RW/Admin/group/Mix/Conflict.status":  kivik.StatusConflict,

		"GetAttachment/RW/group/Admin/foo/NotFound.status": kivik.StatusNotFound,
		"GetAttachment/RW/group/NoAuth.status":             kivik.StatusUnauthorized,

		"GetAttachmentMeta/RW/group/Admin/foo/NotFound.status": kivik.StatusNotFound,
		"GetAttachmentMeta/RW/group/NoAuth.status":             kivik.StatusUnauthorized,

		"PutAttachment/RW/group/Admin/Conflict.status": kivik.StatusInternalServerError, // COUCHDB-3361
		"PutAttachment/RW/group/NoAuth.status":         kivik.StatusUnauthorized,

		// "DeleteAttachment/RW/group/Admin/NotFound.status":  kivik.StatusNotFound, // COUCHDB-3362
		"DeleteAttachment/RW/group/NoAuth.status":       kivik.StatusUnauthorized,
		"DeleteAttachment/RW/group/Admin/NoDoc.status":  kivik.StatusInternalServerError,
		"DeleteAttachment/RW/group/NoAuth/NoDoc.status": kivik.StatusUnauthorized,

		"Put/RW/Admin/group/LeadingUnderscoreInID.status": kivik.StatusBadRequest,
		"Put/RW/Admin/group/Conflict.status":              kivik.StatusConflict,
		"Put/RW/NoAuth/group.status":                      kivik.StatusUnauthorized,
		"Put/RW/NoAuth/group/Conflict.skip":               true,

		"CreateIndex/RW/Admin/group/EmptyIndex.status":    kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/BlankIndex.status":    kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/InvalidIndex.status":  kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/NilIndex.status":      kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/InvalidJSON.status":   kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/EmptyIndex.status":   kivik.StatusUnauthorized,
		"CreateIndex/RW/NoAuth/group/BlankIndex.status":   kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/InvalidIndex.status": kivik.StatusUnauthorized,
		"CreateIndex/RW/NoAuth/group/NilIndex.status":     kivik.StatusUnauthorized,
		"CreateIndex/RW/NoAuth/group/InvalidJSON.status":  kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/Valid.status":        kivik.StatusUnauthorized,

		"GetIndexes.databases":                     []string{"_replicator", "_users", "_global_changes", "chicken", "_duck"},
		"GetIndexes/Admin/_replicator.indexes":     []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/Admin/_users.indexes":          []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/Admin/_global_changes.status":  kivik.StatusForbidden,
		"GetIndexes/Admin/chicken.status":          kivik.StatusNotFound,
		"GetIndexes/Admin/_duck.status":            kivik.StatusForbidden,
		"GetIndexes/NoAuth/_replicator.indexes":    []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/NoAuth/_users.indexes":         []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/NoAuth/_global_changes.status": kivik.StatusForbidden,
		"GetIndexes/NoAuth/chicken.status":         kivik.StatusNotFound,
		"GetIndexes/NoAuth/_duck.status":           kivik.StatusForbidden,
		"GetIndexes/RW/NoAuth.status":              kivik.StatusUnauthorized,

		"DeleteIndex/RW/Admin/group/NotFoundDdoc.status": kivik.StatusNotFound,
		"DeleteIndex/RW/Admin/group/NotFoundName.status": kivik.StatusNotFound,
		"DeleteIndex/RW/NoAuth.status":                   kivik.StatusUnauthorized,

		"GetReplications/NoAuth.status": kivik.StatusUnauthorized,

		"Replicate.NotFoundDB":                                  "http://localhost:5984/foo",
		"Replicate.timeoutSeconds":                              300,
		"Replicate/RW/NoAuth.status":                            kivik.StatusUnauthorized,
		"Replicate/RW/Admin/group/MissingSource/Results.status": kivik.StatusInternalServerError,
		"Replicate/RW/Admin/group/MissingTarget/Results.status": kivik.StatusInternalServerError,

		"Query/RW/group/Admin/WithoutDocs/ScanDoc.status":  kivik.StatusBadRequest,
		"Query/RW/group/NoAuth/WithoutDocs/ScanDoc.status": kivik.StatusBadRequest,
	})
}
