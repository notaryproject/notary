package couchdb

import "fmt"

// Available options
const (
	optionForceCommit = "force_commit"
)

func getAnyKey(i map[string]interface{}) (string, bool) {
	for k := range i {
		return k, true
	}
	return "", false
}

func forceCommit(opts map[string]interface{}) (bool, error) {
	fc, ok := opts[optionForceCommit]
	if !ok {
		return false, nil
	}
	fcBool, ok := fc.(bool)
	if !ok {
		return false, fmt.Errorf("kivik: option '%s' must be bool, not %T", optionForceCommit, fcBool)
	}
	delete(opts, optionForceCommit)
	return fcBool, nil
}
