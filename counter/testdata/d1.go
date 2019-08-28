package testdata

import "log"

func print1() {
	go log.Println("print1")
	log.Println("print1")
}
