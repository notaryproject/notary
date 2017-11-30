package test

import (
	"github.com/flimzy/kivik"
	"github.com/go-kivik/kiviktest"
	"github.com/go-kivik/kiviktest/kt"
)

func registerSuiteCouch16() {
	kiviktest.RegisterSuite(kiviktest.SuiteCouch16, kt.SuiteConfig{
		"AllDBs.expected": []string{"_replicator", "_users"},

		"CreateDB/RW/NoAuth.status":         kivik.StatusUnauthorized,
		"CreateDB/RW/Admin/Recreate.status": kivik.StatusPreconditionFailed,

		"DestroyDB/RW/NoAuth.status":              kivik.StatusUnauthorized,
		"DestroyDB/RW/Admin/NonExistantDB.status": kivik.StatusNotFound,

		"AllDocs.databases":                  []string{"_replicator", "chicken", "_duck"},
		"AllDocs/Admin/_replicator.expected": []string{"_design/_replicator"},
		"AllDocs/Admin/_replicator.offset":   0,
		"AllDocs/Admin/chicken.status":       kivik.StatusNotFound,
		"AllDocs/Admin/_duck.status":         kivik.StatusBadRequest,
		"AllDocs/NoAuth/_replicator.status":  kivik.StatusForbidden,
		"AllDocs/NoAuth/chicken.status":      kivik.StatusNotFound,
		"AllDocs/NoAuth/_duck.status":        kivik.StatusBadRequest,

		"Find.databases":     []string{"_users"},
		"Find.status":        kivik.StatusNotImplemented, // Couchdb 1.6 doesn't support the find interface
		"CreateIndex.status": kivik.StatusNotImplemented, // Couchdb 1.6 doesn't support the find interface
		"GetIndexes.skip":    true,                       // Couchdb 1.6 doesn't support the find interface
		"DeleteIndex.skip":   true,                       // Couchdb 1.6 doesn't support the find interface
		"Explain.databases":  []string{"_users"},
		"Explain.status":     kivik.StatusNotImplemented, // Couchdb 1.6 doesn't support the find interface

		"DBExists.databases":              []string{"_users", "chicken", "_duck"},
		"DBExists/Admin/_users.exists":    true,
		"DBExists/Admin/chicken.exists":   false,
		"DBExists/Admin/_duck.status":     kivik.StatusBadRequest,
		"DBExists/NoAuth/_users.exists":   true,
		"DBExists/NoAuth/chicken.exists":  false,
		"DBExists/NoAuth/_duck.status":    kivik.StatusBadRequest,
		"DBExists/RW/group/Admin.exists":  true,
		"DBExists/RW/group/NoAuth.exists": true,

		"Log/NoAuth.status":                   kivik.StatusUnauthorized,
		"Log/NoAuth/Offset-1000.status":       kivik.StatusBadRequest,
		"Log/Admin/Offset-1000.status":        kivik.StatusBadRequest,
		"Log/Admin/HTTP/NegativeBytes.status": kivik.StatusInternalServerError,
		"Log/Admin/HTTP/TextBytes.status":     kivik.StatusInternalServerError,

		"Version.version":        `^1\.6\.1$`,
		"Version.vendor":         `^The Apache Software Foundation$`,
		"Version.vendor_version": `^1\.6\.1$`,

		"Get/RW/group/Admin/bogus.status":  kivik.StatusNotFound,
		"Get/RW/group/NoAuth/bogus.status": kivik.StatusNotFound,

		"Rev/RW/group/Admin/bogus.status":  kivik.StatusNotFound,
		"Rev/RW/group/NoAuth/bogus.status": kivik.StatusNotFound,

		"Flush.databases":                     []string{"_users", "chicken", "_duck"},
		"Flush/Admin/chicken/DoFlush.status":  kivik.StatusNotFound,
		"Flush/Admin/_duck/DoFlush.status":    kivik.StatusBadRequest,
		"Flush/NoAuth/chicken/DoFlush.status": kivik.StatusNotFound,
		"Flush/NoAuth/_duck/DoFlush.status":   kivik.StatusBadRequest,

		"Delete/RW/Admin/group/MissingDoc.status":        kivik.StatusNotFound,
		"Delete/RW/Admin/group/InvalidRevFormat.status":  kivik.StatusBadRequest,
		"Delete/RW/Admin/group/WrongRev.status":          kivik.StatusConflict,
		"Delete/RW/NoAuth/group/MissingDoc.status":       kivik.StatusNotFound,
		"Delete/RW/NoAuth/group/InvalidRevFormat.status": kivik.StatusBadRequest,
		"Delete/RW/NoAuth/group/WrongRev.status":         kivik.StatusConflict,
		"Delete/RW/NoAuth/group/DesignDoc.status":        kivik.StatusUnauthorized,

		"Session/Get/Admin.info.authentication_handlers":  "oauth,cookie,default",
		"Session/Get/Admin.info.authentication_db":        "_users",
		"Session/Get/Admin.info.authenticated":            "cookie",
		"Session/Get/Admin.userCtx.roles":                 "_admin",
		"Session/Get/Admin.ok":                            "true",
		"Session/Get/NoAuth.info.authentication_handlers": "oauth,cookie,default",
		"Session/Get/NoAuth.info.authentication_db":       "_users",
		"Session/Get/NoAuth.info.authenticated":           "",
		"Session/Get/NoAuth.userCtx.roles":                "",
		"Session/Get/NoAuth.ok":                           "true",

		"Session/Post/EmptyJSON.status":                             kivik.StatusBadRequest,
		"Session/Post/BogusTypeJSON.status":                         kivik.StatusUnauthorized,
		"Session/Post/BogusTypeForm.status":                         kivik.StatusUnauthorized,
		"Session/Post/EmptyForm.status":                             kivik.StatusUnauthorized,
		"Session/Post/BadJSON.status":                               kivik.StatusBadRequest,
		"Session/Post/BadForm.status":                               kivik.StatusUnauthorized,
		"Session/Post/MeaninglessJSON.status":                       kivik.StatusInternalServerError,
		"Session/Post/MeaninglessForm.status":                       kivik.StatusUnauthorized,
		"Session/Post/GoodJSON.status":                              kivik.StatusUnauthorized,
		"Session/Post/BadQueryParam.status":                         kivik.StatusUnauthorized,
		"Session/Post/BadCredsJSON.status":                          kivik.StatusUnauthorized,
		"Session/Post/BadCredsForm.status":                          kivik.StatusUnauthorized,
		"Session/Post/GoodCredsJSONRemoteRedirHeaderInjection.skip": true, // CouchDB allows header injection
		"Session/Post/GoodCredsJSONRemoteRedirInvalidURL.skip":      true, // CouchDB doesn't sanitize the Location value, so sends unparseable headers.

		"Stats.databases":             []string{"_users", "chicken", "_duck"},
		"Stats/Admin/chicken.status":  kivik.StatusNotFound,
		"Stats/Admin/_duck.status":    kivik.StatusBadRequest,
		"Stats/NoAuth/chicken.status": kivik.StatusNotFound,
		"Stats/NoAuth/_duck.status":   kivik.StatusBadRequest,

		"Compact/RW/NoAuth.status": kivik.StatusUnauthorized,

		"ViewCleanup/RW/NoAuth.status": kivik.StatusUnauthorized,

		"Security.databases":              []string{"_replicator", "_users", "chicken", "_duck"},
		"Security/Admin/chicken.status":   kivik.StatusNotFound,
		"Security/Admin/_duck.status":     kivik.StatusBadRequest,
		"Security/NoAuth/chicken.status":  kivik.StatusNotFound,
		"Security/NoAuth/_duck.status":    kivik.StatusBadRequest,
		"Security/RW/group/NoAuth.status": kivik.StatusUnauthorized,

		"SetSecurity/RW/Admin/NotExists.status":  kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/NotExists.status": kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/Exists.status":    kivik.StatusUnauthorized,

		"DBUpdates/RW/NoAuth.status": kivik.StatusUnauthorized,

		"BulkDocs/RW/NoAuth/group/Mix/Conflict.status": kivik.StatusConflict,
		"BulkDocs/RW/Admin/group/Mix/Conflict.status":  kivik.StatusConflict,

		"GetAttachment/RW/group/Admin/foo/NotFound.status":  kivik.StatusNotFound,
		"GetAttachment/RW/group/NoAuth/foo/NotFound.status": kivik.StatusNotFound,

		"GetAttachmentMeta/RW/group/Admin/foo/NotFound.status":  kivik.StatusNotFound,
		"GetAttachmentMeta/RW/group/NoAuth/foo/NotFound.status": kivik.StatusNotFound,

		"PutAttachment/RW/group/Admin/Conflict.status":         kivik.StatusConflict,
		"PutAttachment/RW/group/NoAuth/Conflict.status":        kivik.StatusConflict,
		"PutAttachment/RW/group/NoAuth/UpdateDesignDoc.status": kivik.StatusUnauthorized,
		"PutAttachment/RW/group/NoAuth/CreateDesignDoc.status": kivik.StatusUnauthorized,

		// "DeleteAttachment/RW/group/Admin/NotFound.status":  kivik.StatusNotFound, // COUCHDB-3362
		// "DeleteAttachment/RW/group/NoAuth/NotFound.status": kivik.StatusNotFound, // COUCHDB-3362
		"DeleteAttachment/RW/group/Admin/NoDoc.status":      kivik.StatusConflict,
		"DeleteAttachment/RW/group/NoAuth/NoDoc.status":     kivik.StatusConflict,
		"DeleteAttachment/RW/group/NoAuth/DesignDoc.status": kivik.StatusUnauthorized,

		"Put/RW/Admin/group/LeadingUnderscoreInID.status":  kivik.StatusBadRequest,
		"Put/RW/Admin/group/Conflict.status":               kivik.StatusConflict,
		"Put/RW/NoAuth/group/LeadingUnderscoreInID.status": kivik.StatusBadRequest,
		"Put/RW/NoAuth/group/DesignDoc.status":             kivik.StatusUnauthorized,
		"Put/RW/NoAuth/group/Conflict.status":              kivik.StatusConflict,

		"GetReplications/NoAuth.status": kivik.StatusForbidden,

		"Replicate.NotFoundDB":                                  "http://localhost:5984/foo",
		"Replicate.timeoutSeconds":                              120,
		"Replicate.prefix":                                      "none",
		"Replicate/RW/NoAuth.status":                            kivik.StatusForbidden,
		"Replicate/RW/Admin/group/MissingSource/Results.status": kivik.StatusNotFound,
		"Replicate/RW/Admin/group/MissingTarget/Results.status": kivik.StatusNotFound,

		"Query/RW/group/Admin/WithoutDocs/ScanDoc.status":  kivik.StatusBadRequest,
		"Query/RW/group/NoAuth/WithoutDocs/ScanDoc.status": kivik.StatusBadRequest,
	})
}
