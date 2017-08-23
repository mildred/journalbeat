package main

import (
	"context"
	"flag"
	"github.com/mheese/journalbeat/journalclient"
	"log"
	"sync"
)

func main() {
	url := flag.String("gateway", "http://localhost:19531", "systemd-journal-gatewayd address")
	curs := flag.String("cursor", "", "current cursor")
	flag.Parse()

	var cursor string = *curs
	var wg sync.WaitGroup
	ctx := context.Background()
	for msg := range journalclient.Stream(ctx, &wg, *url, &cursor) {
		for k, v := range msg {
			log.Printf("%s=%s\n", k, v)
		}
		log.Println()
	}
}
