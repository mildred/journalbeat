package journalclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

func Stream(ctx context.Context, wg *sync.WaitGroup, address string, cursor *string) <-chan map[string][]byte {
	c := make(chan map[string][]byte)
	wg.Add(1)
	go stream(ctx, wg, c, address, cursor)
	return c
}

func stream(ctx context.Context, wg *sync.WaitGroup, c chan<- map[string][]byte, address string, cursor *string) {
	defer wg.Done()
	var client http.Client

	for ctx.Err() == nil {
		req, err := http.NewRequest(http.MethodGet, address+"/entries?follow", nil)
		if err != nil {
			log.Print(err)
			sleep(ctx)
			continue
		}

		req.Header.Set("Accept", "application/vnd.fdo.journal")
		log.Print("Accept: application/vnd.fdo.journal")
		if *cursor != "" {
			req.Header.Set("Range", "entries="+*cursor+":1:")
			log.Printf("Range: entries=%s:1:", *cursor)
		}
		req.WithContext(ctx)

		log.Printf("request: %s", req.URL.String())

		res, err := client.Do(req)
		if err != nil {
			log.Print(err)
			sleep(ctx)
			continue
		}

		err = readMessages(ctx, wg, c, res.Body, cursor)
		if err != nil {
			log.Print(err)
			sleep(ctx)
			continue
		}

		sleep(ctx)
	}
}

func readMessages(ctx context.Context, wg *sync.WaitGroup, c chan<- map[string][]byte, body io.ReadCloser, cursor *string) error {
	defer body.Close()
	stream := bufio.NewReaderSize(body, 16)

	for ctx.Err() == nil {
		cur := map[string][]byte{}
		for ctx.Err() == nil {
			line, err := stream.ReadBytes('\n')
			if err != nil {
				return err
			}
			log.Printf("read line %#v", string(line))

			if string(line) == "\n" {
				break
			}

			if bytes.HasSuffix(line, []byte{'\n'}) {
				line = line[0 : len(line)-1]
			}

			i := bytes.Index(line, []byte{'='})
			if i == -1 {
				// read binary data
				var size uint64
				key := line
				err := binary.Read(stream, binary.LittleEndian, &size)
				if err != nil {
					return err
				}
				val := make([]byte, size)
				_, err = io.ReadFull(stream, val)
				if err != nil {
					return err
				}
				_, err = stream.ReadByte()
				if err != nil {
					return err
				}
				cur[string(key)] = val
			} else {
				key := line[0:i]
				val := line[i+1 : len(line)]
				cur[string(key)] = val
			}
		}
		c <- cur
		if curs, ok := cur["__CURSOR"]; ok {
			*cursor = string(curs)
		}
	}

	return nil
}

func sleep(ctx context.Context) {
	ctx2, _ := context.WithTimeout(ctx, 10*time.Second)
	<-ctx2.Done()
}
