// +build !js

package test

import (
	"testing"

	_ "github.com/go-kivik/couchdb"
	"github.com/go-kivik/kiviktest"
)

func TestCouch16(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch16, "KIVIK_TEST_DSN_COUCH16", t)
}

func TestCloudant(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCloudant, "KIVIK_TEST_DSN_CLOUDANT", t)
}

func TestCouch20(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch20, "KIVIK_TEST_DSN_COUCH20", t)
}

func TestCouch21(t *testing.T) {
	kiviktest.DoTest(kiviktest.SuiteCouch21, "KIVIK_TEST_DSN_COUCH21", t)
}
