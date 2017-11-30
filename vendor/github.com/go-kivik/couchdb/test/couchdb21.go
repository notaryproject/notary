package test

import (
	"github.com/flimzy/kivik"
	"github.com/go-kivik/kiviktest"
	"github.com/go-kivik/kiviktest/kt"
)

func registerSuiteCouch21() {
	kiviktest.RegisterSuite(kiviktest.SuiteCouch21, kt.SuiteConfig{
		"AllDBs.expected": []string{"_global_changes", "_replicator", "_users"},

		"CreateDB/RW/NoAuth.status":         kivik.StatusUnauthorized,
		"CreateDB/RW/Admin/Recreate.status": kivik.StatusPreconditionFailed,

		"DestroyDB/RW/NoAuth.status":              kivik.StatusUnauthorized,
		"DestroyDB/RW/Admin/NonExistantDB.status": kivik.StatusNotFound,

		"AllDocs.databases":                 []string{"chicken", "_duck"},
		"AllDocs/Admin/_replicator.offset":  0,
		"AllDocs/Admin/chicken.status":      kivik.StatusNotFound,
		"AllDocs/Admin/_duck.status":        kivik.StatusNotFound,
		"AllDocs/NoAuth/_replicator.status": kivik.StatusUnauthorized,
		"AllDocs/NoAuth/chicken.status":     kivik.StatusNotFound,
		"AllDocs/NoAuth/_duck.status":       kivik.StatusUnauthorized,

		"Find.databases":                       []string{"chicken", "_duck"},
		"Find/Admin/chicken.status":            kivik.StatusNotFound,
		"Find/Admin/_duck.status":              kivik.StatusNotFound,
		"Find/NoAuth/chicken.status":           kivik.StatusNotFound,
		"Find/NoAuth/_duck.status":             kivik.StatusUnauthorized,
		"Find/RW/group/Admin/Warning.warning":  "no matching index found, create an index to optimize query time",
		"Find/RW/group/NoAuth/Warning.warning": "no matching index found, create an index to optimize query time",

		"Explain.databases":             []string{"chicken", "_duck"},
		"Explain/Admin/chicken.status":  kivik.StatusNotFound,
		"Explain/Admin/_duck.status":    kivik.StatusNotFound,
		"Explain/NoAuth/chicken.status": kivik.StatusNotFound,
		"Explain/NoAuth/_duck.status":   kivik.StatusUnauthorized,
		"Explain.plan": &kivik.QueryPlan{
			Index: map[string]interface{}{
				"ddoc": nil,
				"name": "_all_docs",
				"type": "special",
				"def":  map[string]interface{}{"fields": []interface{}{map[string]string{"_id": "asc"}}},
			},
			Selector: map[string]interface{}{"_id": map[string]interface{}{"$gt": nil}},
			Options: map[string]interface{}{
				"bookmark":        "nil",
				"conflicts":       false,
				"execution_stats": false,
				"r":               []int{49},
				"sort":            map[string]interface{}{},
				"use_index":       []interface{}{},
				"stable":          false,
				"stale":           false,
				"update":          true,
				"skip":            0,
				"limit":           25,
				"fields":          "all_fields",
			},
			Range: nil,
			Limit: 25,
		},

		"DBExists.databases":              []string{"_users", "chicken", "_duck"},
		"DBExists/Admin/_users.exists":    true,
		"DBExists/Admin/chicken.exists":   false,
		"DBExists/Admin/_duck.exists":     false,
		"DBExists/NoAuth/_users.exists":   true,
		"DBExists/NoAuth/chicken.exists":  false,
		"DBExists/NoAuth/_duck.status":    kivik.StatusUnauthorized,
		"DBExists/RW/group/Admin.exists":  true,
		"DBExists/RW/group/NoAuth.exists": true,

		"Log.skip": true, // This was removed in CouchDB 2.0

		"Version.version":        `^2\.1\.`,
		"Version.vendor":         `^The Apache Software Foundation$`,
		"Version.vendor_version": ``, // CouchDB 2.0+ no longer has a vendor version

		"Get/RW/group/Admin/bogus.status":  kivik.StatusNotFound,
		"Get/RW/group/NoAuth/bogus.status": kivik.StatusNotFound,

		"Rev/RW/group/Admin/bogus.status":  kivik.StatusNotFound,
		"Rev/RW/group/NoAuth/bogus.status": kivik.StatusNotFound,

		"Flush.databases":                     []string{"_users", "chicken", "_duck"},
		"Flush/NoAuth/chicken/DoFlush.status": kivik.StatusNotFound,
		"Flush/Admin/chicken/DoFlush.status":  kivik.StatusNotFound,
		"Flush/Admin/_duck/DoFlush.status":    kivik.StatusNotFound,
		"Flush/NoAuth/_duck/DoFlush.status":   kivik.StatusUnauthorized,

		"Delete/RW/Admin/group/MissingDoc.status":        kivik.StatusNotFound,
		"Delete/RW/Admin/group/InvalidRevFormat.status":  kivik.StatusBadRequest,
		"Delete/RW/Admin/group/WrongRev.status":          kivik.StatusConflict,
		"Delete/RW/NoAuth/group/MissingDoc.status":       kivik.StatusNotFound,
		"Delete/RW/NoAuth/group/InvalidRevFormat.status": kivik.StatusBadRequest,
		"Delete/RW/NoAuth/group/WrongRev.status":         kivik.StatusConflict,
		"Delete/RW/NoAuth/group/DesignDoc.status":        kivik.StatusUnauthorized,

		"Session/Get/Admin.info.authentication_handlers":  "cookie,default",
		"Session/Get/Admin.info.authentication_db":        "_users",
		"Session/Get/Admin.info.authenticated":            "cookie",
		"Session/Get/Admin.userCtx.roles":                 "_admin",
		"Session/Get/Admin.ok":                            "true",
		"Session/Get/NoAuth.info.authentication_handlers": "cookie,default",
		"Session/Get/NoAuth.info.authentication_db":       "_users",
		"Session/Get/NoAuth.info.authenticated":           "",
		"Session/Get/NoAuth.userCtx.roles":                "",
		"Session/Get/NoAuth.ok":                           "true",

		"Session/Post/EmptyJSON.status":                             kivik.StatusBadRequest,
		"Session/Post/BogusTypeJSON.status":                         kivik.StatusBadRequest,
		"Session/Post/BogusTypeForm.status":                         kivik.StatusBadRequest,
		"Session/Post/EmptyForm.status":                             kivik.StatusBadRequest,
		"Session/Post/BadJSON.status":                               kivik.StatusBadRequest,
		"Session/Post/BadForm.status":                               kivik.StatusBadRequest,
		"Session/Post/MeaninglessJSON.status":                       kivik.StatusInternalServerError,
		"Session/Post/MeaninglessForm.status":                       kivik.StatusBadRequest,
		"Session/Post/GoodJSON.status":                              kivik.StatusUnauthorized,
		"Session/Post/BadQueryParam.status":                         kivik.StatusUnauthorized,
		"Session/Post/BadCredsJSON.status":                          kivik.StatusUnauthorized,
		"Session/Post/BadCredsForm.status":                          kivik.StatusUnauthorized,
		"Session/Post/GoodCredsJSONRemoteRedirAbsolute.status":      kivik.StatusBadRequest, // As of 2.1.1 all redirect paths must begin with '/'
		"Session/Post/GoodCredsJSONRedirEmpty.status":               kivik.StatusBadRequest, // As of 2.1.1 all redirect paths must begin with '/'
		"Session/Post/GoodCredsJSONRedirRelativeNoSlash.status":     kivik.StatusBadRequest, // As of 2.1.1 all redirect paths must begin with '/'
		"Session/Post/GoodCredsJSONRemoteRedirHeaderInjection.skip": true,                   // CouchDB allows header injection
		"Session/Post/GoodCredsJSONRemoteRedirInvalidURL.skip":      true,                   // CouchDB doesn't sanitize the Location value, so sends unparseable headers.

		"Stats.databases":             []string{"_users", "chicken", "_duck"},
		"Stats/Admin/chicken.status":  kivik.StatusNotFound,
		"Stats/Admin/_duck.status":    kivik.StatusNotFound,
		"Stats/NoAuth/chicken.status": kivik.StatusNotFound,
		"Stats/NoAuth/_duck.status":   kivik.StatusUnauthorized,

		"Compact.skip":             false,
		"Compact/RW/NoAuth.status": kivik.StatusUnauthorized,

		"Security.databases":                     []string{"_replicator", "_users", "_global_changes", "chicken", "_duck"},
		"Security/Admin/chicken.status":          kivik.StatusNotFound,
		"Security/Admin/_duck.status":            kivik.StatusNotFound,
		"Security/NoAuth/_global_changes.status": kivik.StatusUnauthorized,
		"Security/NoAuth/chicken.status":         kivik.StatusNotFound,
		"Security/NoAuth/_duck.status":           kivik.StatusUnauthorized,
		"Security/RW/group/NoAuth.status":        kivik.StatusUnauthorized,

		"SetSecurity/RW/Admin/NotExists.status":  kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/NotExists.status": kivik.StatusNotFound,
		"SetSecurity/RW/NoAuth/Exists.status":    kivik.StatusInternalServerError, // Can you say WTF?

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

		"CreateIndex/RW/Admin/group/EmptyIndex.status":    kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/BlankIndex.status":    kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/InvalidIndex.status":  kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/NilIndex.status":      kivik.StatusBadRequest,
		"CreateIndex/RW/Admin/group/InvalidJSON.status":   kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/EmptyIndex.status":   kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/BlankIndex.status":   kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/InvalidIndex.status": kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/NilIndex.status":     kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/InvalidJSON.status":  kivik.StatusBadRequest,
		"CreateIndex/RW/NoAuth/group/Valid.status":        kivik.StatusInternalServerError, // COUCHDB-3374

		"GetIndexes.databases":                     []string{"_replicator", "_users", "_global_changes", "chicken", "_duck"},
		"GetIndexes/Admin/_replicator.indexes":     []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/Admin/_users.indexes":          []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/Admin/_global_changes.indexes": []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/Admin/chicken.status":          kivik.StatusNotFound,
		"GetIndexes/Admin/_duck.status":            kivik.StatusNotFound,
		"GetIndexes/NoAuth/_replicator.indexes":    []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/NoAuth/_users.indexes":         []kivik.Index{kt.AllDocsIndex},
		"GetIndexes/NoAuth/_global_changes.status": kivik.StatusUnauthorized,
		"GetIndexes/NoAuth/chicken.status":         kivik.StatusNotFound,
		"GetIndexes/NoAuth/_duck.status":           kivik.StatusUnauthorized,
		"GetIndexes/RW.indexes": []kivik.Index{kt.AllDocsIndex,
			{
				DesignDoc: "_design/foo",
				Name:      "bar",
				Type:      "json",
				Definition: map[string]interface{}{
					"fields": []map[string]string{
						{"foo": "asc"},
					},
					"partial_filter_selector": map[string]string{},
				},
			},
		},

		"DeleteIndex/RW/Admin/group/NotFoundDdoc.status":  kivik.StatusNotFound,
		"DeleteIndex/RW/Admin/group/NotFoundName.status":  kivik.StatusNotFound,
		"DeleteIndex/RW/NoAuth/group/NotFoundDdoc.status": kivik.StatusNotFound,
		"DeleteIndex/RW/NoAuth/group/NotFoundName.status": kivik.StatusNotFound,

		"GetReplications/NoAuth.status": kivik.StatusUnauthorized,

		"Replicate.NotFoundDB":                                  "http://localhost:5984/foo",
		"Replicate.timeoutSeconds":                              60,
		"Replicate.prefix":                                      "http://localhost:5984/",
		"Replicate/RW/NoAuth.status":                            kivik.StatusForbidden,
		"Replicate/RW/Admin/group/MissingSource/Results.status": kivik.StatusNotFound,
		"Replicate/RW/Admin/group/MissingTarget/Results.status": kivik.StatusNotFound,

		"Query/RW/group/Admin/WithoutDocs/ScanDoc.status":  kivik.StatusBadRequest,
		"Query/RW/group/NoAuth/WithoutDocs/ScanDoc.status": kivik.StatusBadRequest,

		"ViewCleanup/RW/NoAuth.status": kivik.StatusUnauthorized,
	})
}
