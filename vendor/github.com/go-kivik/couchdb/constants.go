package couchdb

const (
	// OptionFullCommit is the option key used to set the `X-Couch-Full-Commit`
	// header in the request when set to true.
	//
	// Example:
	//
	//    db.Put(ctx, "doc_id", doc, kivik.Options{couchdb.OptionFullCommit: true})
	OptionFullCommit = "X-Couch-Full-Commit"

	// OptionIfNoneMatch sets the If-None-Match header on the request.
	//
	// Example:
	//
	//    row, err := db.Get(ctx, "doc_id", kivik.Options(couchdb.OptionIfNoneMatch: "1-xxx"))
	OptionIfNoneMatch = "If-None-Match"
)

// optionForceCommit is an unfortunately mispelled version of "full-commit",
// retained for backward compatibility.
const optionForceCommit = "force_commit"
