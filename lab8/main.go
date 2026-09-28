package main

import (
	"log"

	"github.com/Di0ks/go-programming-labs/lab8/restapi"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	cfg := restapi.NewDbConfig()
	restapi.RunApi("127.0.0.1", 8080, &cfg)
}
