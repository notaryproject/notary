package chttp

import (
	"net/url"
	"strings"
)

const (
	prefixDesign = "_design/"
	prefixLocal  = "_local/"
)

// EncodeDocID encodes a document ID according to CouchDB's path encoding rules.
//
// In particular:
// -  '_design/' and '_local/' prefixes are unaltered.
// - The rest of the docID is Query-URL encoded (despite being part of the path)
func EncodeDocID(docID string) string {
	for _, prefix := range []string{prefixDesign, prefixLocal} {
		if strings.HasPrefix(docID, prefix) {
			return prefix + url.QueryEscape(strings.TrimPrefix(docID, prefix))
		}
	}
	return url.QueryEscape(docID)
}
