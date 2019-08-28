package testdata

import "log"

func print2() {
	log.Println("print2")
	log.Println("print2")
	print3()
}

func print3() {
	log.Println("print3")
	log.Println("print3")
	log.Println("print3")
}

func print4(f1, f2 func()) {
	log.Println("print4")
	f1()
	f2()
	log.Println("print4")
}

// test11111
func print5() {
	print4(func() {
		log.Println("print5-1")
		log.Println("print5-1")
	}, func() {
		log.Println("print5-2")
		log.Println("print5-2")
	})
}
