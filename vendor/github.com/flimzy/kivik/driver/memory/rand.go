// +build go1.8

package memory

import "fmt"

func randStr() string {
	rndMU.Lock()
	s := fmt.Sprintf("%016x%016x", rnd.Uint64(), rnd.Uint64())
	rndMU.Unlock()
	return s
}
