package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println("Query target:", os.Args[1])
	} else {
		fmt.Println("No target provided, starting server")
	}
}
