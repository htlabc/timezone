package consensus

import (
	"fmt"
	"os"
)

var filename string = ""

func Write(data []byte) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("os.create file err %v", err)
		return
	}
	file.Write(data)
}
