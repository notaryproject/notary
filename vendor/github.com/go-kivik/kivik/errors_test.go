package kivik

import (
	"errors"
	"testing"

	kerrors "github.com/go-kivik/kivik/errors"
)

func TestStatusCoder(t *testing.T) {
	type scTest struct {
		Name     string
		Err      error
		Expected int
	}
	tests := []scTest{
		{
			Name:     "nil",
			Expected: 0,
		},
		{
			Name:     "Standard error",
			Err:      errors.New("foo"),
			Expected: 500,
		},
		{
			Name:     "StatusCoder",
			Err:      kerrors.Status(400, "bad request"),
			Expected: 400,
		},
	}
	for _, test := range tests {
		func(test scTest) {
			t.Run(test.Name, func(t *testing.T) {
				result := StatusCode(test.Err)
				if result != test.Expected {
					t.Errorf("Unexpected result. Expected %d, got %d", test.Expected, result)
				}
			})
		}(test)
	}
}

type testReasoner int

func (tr testReasoner) Reason() string { return "reason" }
func (tr testReasoner) Error() string  { return "error" }

func TestReasoner(t *testing.T) {
	type rTest struct {
		Name     string
		Err      error
		Expected string
	}
	tests := []rTest{
		{
			Name:     "NilError",
			Err:      nil,
			Expected: "",
		},
		{
			Name:     "NonReasoner",
			Err:      errors.New("test error"),
			Expected: "test error",
		},
		{
			Name:     "Reasoner",
			Err:      testReasoner(500),
			Expected: "reason",
		},
	}
	for _, test := range tests {
		func(test rTest) {
			t.Run(test.Name, func(t *testing.T) {
				result := Reason(test.Err)
				if result != test.Expected {
					t.Errorf("Expected: %s\n  Actual: %s", test.Expected, result)
				}
			})
		}(test)
	}
}
