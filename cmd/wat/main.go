package main

import (
	"fmt"
	"os"

	"github.com/windmilleng/wat/cli/wat"
)

func main() {
	err := wat.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
