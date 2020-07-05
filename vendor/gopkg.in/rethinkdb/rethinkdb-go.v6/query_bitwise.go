package rethinkdb

import (
	p "gopkg.in/rethinkdb/rethinkdb-go.v6/ql2"
)

// Rethinkdb proposal: https://github.com/rethinkdb/rethinkdb/pull/6534

// Or performs a bitwise And.
func (t Term) BitAnd(args ...interface{}) Term {
	return constructMethodTerm(t, "BitAnd", p.Term_BIT_AND, args, map[string]interface{}{})
}

// Or performs a bitwise And.
func BitAnd(args ...interface{}) Term {
	return constructRootTerm("BitAnd", p.Term_BIT_AND, args, map[string]interface{}{})
}

// Or performs a bitwise Or.
func (t Term) BitOr(args ...interface{}) Term {
	return constructMethodTerm(t, "BitOr", p.Term_BIT_OR, args, map[string]interface{}{})
}

// Or performs a bitwise Or.
func BitOr(args ...interface{}) Term {
	return constructRootTerm("BitOr", p.Term_BIT_OR, args, map[string]interface{}{})
}

// Or performs a bitwise XOR.
func (t Term) BitXor(args ...interface{}) Term {
	return constructMethodTerm(t, "BitXor", p.Term_BIT_XOR, args, map[string]interface{}{})
}

// Or performs a bitwise XOR.
func BitXor(args ...interface{}) Term {
	return constructRootTerm("BitXor", p.Term_BIT_XOR, args, map[string]interface{}{})
}

// Or performs a bitwise complement.
func (t Term) BitNot() Term {
	return constructMethodTerm(t, "BitNot", p.Term_BIT_NOT, []interface{}{}, map[string]interface{}{})
}

// Or performs a bitwise complement.
func BitNot(arg interface{}) Term {
	return constructRootTerm("BitNot", p.Term_BIT_NOT, []interface{}{arg}, map[string]interface{}{})
}

// Or performs a bitwise shift arithmetic left.
func (t Term) BitSal(args ...interface{}) Term {
	return constructMethodTerm(t, "BitSal", p.Term_BIT_SAL, args, map[string]interface{}{})
}

// Or performs a bitwise shift arithmetic left.
func BitSal(args ...interface{}) Term {
	return constructRootTerm("BitSal", p.Term_BIT_SAL, args, map[string]interface{}{})
}

//// Or performs a bitwise left shift.
//func (t Term) BitShl(args ...interface{}) Term {
//	return constructMethodTerm(t, "BitShl", p.Term_BIT_SAL, args, map[string]interface{}{})
//}
//
//// Or performs a bitwise left shift.
//func BitShl(args ...interface{}) Term {
//	return constructRootTerm("BitShl", p.Term_BIT_SAL, args, map[string]interface{}{})
//}

// Or performs a bitwise shift arithmetic right.
func (t Term) BitSar(args ...interface{}) Term {
	return constructMethodTerm(t, "BitSar", p.Term_BIT_SAR, args, map[string]interface{}{})
}

// Or performs a bitwise shift arithmetic right.
func BitSar(args ...interface{}) Term {
	return constructRootTerm("BitSar", p.Term_BIT_SAR, args, map[string]interface{}{})
}

//// Or performs a bitwise right shift.
//func (t Term) BitShr(args ...interface{}) Term {
//	return constructMethodTerm(t, "BitShr", p.Term_BIT_SHR, args, map[string]interface{}{})
//}
//
//// Or performs a bitwise right shift.
//func BitShr(args ...interface{}) Term {
//	return constructRootTerm("BitShr", p.Term_BIT_SHR, args, map[string]interface{}{})
//}
