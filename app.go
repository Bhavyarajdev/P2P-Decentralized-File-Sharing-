// go:build js && wasm
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"syscall/js"
	"time"
)

func extractFilename(this js.Value, args []js.Value) any {
	path := args[0].String()
	pathSplit := strings.Split(path, "\\")
	return pathSplit[2]
}


func main() {
	c := make(chan struct{})

	
	js.Global().Set("extractFname", js.FuncOf(extractFilename))
	document := js.Global().Get("document")

	go func() {
		for {
			time.Sleep(1 * time.Second)
			resp, err := http.Get("http://localhost:43480/protoMsg?msg=progress")
			if err != nil {
				log.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				log.Fatal(resp.Status)
			}
			defer resp.Body.Close()
			read, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			js.Global().Call("update", string(read))
		}
	}()
	fmt.Println(document)
	<-c
}
