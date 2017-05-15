//libadd.go
package main

import "C"

//export add
func add(left, right int) int {
	return left + right
}

//export list
func list() []string {
	return []string{
		"lol",
	}
}

func main() {
}
