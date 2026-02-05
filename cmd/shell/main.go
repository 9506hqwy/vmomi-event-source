package main

import (
	"fmt"

	"github.com/9506hqwy/template-go-module/pkg/example"
)

var version = "<version>"
var commit = "<commit>"

func main() {
	//revive:disable:add-constant
	ret := example.Add(2, 5)

	_, err := fmt.Printf("result = %d\n", ret)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Printf("version = %s, commit = %s\n", version, commit)
	if err != nil {
		panic(err)
	}
}
