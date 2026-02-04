package main

import "os"

func main() {
	root := NewRoot()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
