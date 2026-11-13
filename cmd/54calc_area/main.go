package main

import "os"

func main() {
	const filename = ""

	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

}
