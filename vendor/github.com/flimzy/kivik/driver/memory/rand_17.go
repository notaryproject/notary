// +build go1.7,!go1.8

package memory

import "fmt"

func randStr() string {
	rndMU.Lock()
	s := fmt.Sprintf("%012x%012x%08x", rnd.Int63n(1<<48), rnd.Int63n(1<<48), rnd.Int63n(1<<32))
	rndMU.Unlock()
	return s
}
