package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

func main() {
	src := []byte(`
	package main
	
	import "fmt"
	
	func main() {
		fmt.Println("hello world")
	}
	`)
	if err := ioutil.WriteFile("hw/test.go", src, 0644); err != nil {
		panic(err)
	}

	out, err := exec.Command("sh", "-c", "go run hw/test.go").Output()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

}
