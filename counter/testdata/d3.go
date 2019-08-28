package testdata

import "log"

func print6() {
	data := make([]byte, 6)
	log.Println("print6", cap(data))
}
