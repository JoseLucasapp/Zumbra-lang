package main

import "fmt"

func main() {
	var x = 0
	fmt.Println(x)
	for x < 3 {
		fmt.Println(x)
		x = x + 1
		fmt.Println(x)
	}
	fmt.Println(x)
}
