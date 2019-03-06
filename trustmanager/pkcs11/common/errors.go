package common

import "fmt"

// ErrNoKeysFound is returned when there are no keys in the HardwareStore
type ErrNoKeysFound struct {
	HSM string
}

func (e ErrNoKeysFound) Error() string {
	return fmt.Sprintf("no keys found in %s", e.HSM)
}
