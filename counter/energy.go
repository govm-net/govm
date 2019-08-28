package counter

import (
	"sync"
)

const defaultEnergy = 1 << 20

var mu sync.Mutex
var energy uint64 = defaultEnergy
var used uint64

// SetEnergy set energy
func SetEnergy(n uint64) {
	mu.Lock()
	defer mu.Unlock()
	if energy > defaultEnergy {
		panic(energy)
	}
	energy += n
	if energy <= defaultEnergy {
		panic(energy)
	}
	energy -= defaultEnergy
}

// ConsumeEnergy consume energy
func ConsumeEnergy(n uint64) {
	if n == 0 {
		n = 10
	}
	mu.Lock()
	defer mu.Unlock()
	used += used / 100000
	used += n
	if used > energy {
		panic("not enough energy")
	}
}
