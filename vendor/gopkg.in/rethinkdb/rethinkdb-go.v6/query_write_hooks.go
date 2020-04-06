package rethinkdb

import (
	p "gopkg.in/rethinkdb/rethinkdb-go.v6/ql2"
)

// Rethinkdb proposal: https://github.com/rethinkdb/rethinkdb/issues/5813

// WriteHookFunc called by rethinkdb when document is changed.
// id, oldVal or newVal can be null (test with Branch).
type WriteHookFunc func(id Term, oldVal Term, newVal Term) Term

// SetWriteHook sets function that will be called each time document in a table is being
// inserted, updated, replaced or deleted.
func (t Term) SetWriteHook(hookFunc WriteHookFunc) Term {
	var f interface{} = nil
	if hookFunc != nil {
		f = funcWrap(hookFunc)
	}
	return constructMethodTerm(t, "SetWriteHook", p.Term_SET_WRITE_HOOK, []interface{}{f}, map[string]interface{}{})
}

// WriteHookInfo is a return type of GetWriteHook func.
type WriteHookInfo struct {
	Function []byte `gorethink:"function,omitempty"`
	Query    string `gorethink:"query,omitempty"`
}

// GetWriteHook reads write hook associated with table.
// Use WriteHookInfo and ReadOne to get results.
func (t Term) GetWriteHook() Term {
	return constructMethodTerm(t, "GetWriteHook", p.Term_GET_WRITE_HOOK, []interface{}{}, map[string]interface{}{})
}
