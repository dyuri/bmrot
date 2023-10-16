package main

import (
	"fmt"
	"log"
	"os"
)

func printDescriptor(desc *Descriptor) {
	fmt.Printf("%s\n", desc)
}

func main() {
	// get file name from command line
	if len(os.Args) < 2 {
		fmt.Println("Usage: bmrot <filename>")
		os.Exit(1)
	}

	filename := os.Args[1]
	desc, err := LoadDescriptor(filename)
	if err != nil {
		log.Fatal(err)
	}

	desc.Rotate()

	printDescriptor(desc)
}
