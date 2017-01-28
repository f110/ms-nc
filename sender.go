package main

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/time/rate"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	// ReadBufferSize = 5120000
	ReadBufferSize = 512
	MaxThreads     = 128
)

func NewSender(addr string, r io.Reader, mBytes float32) {
	sem := make(chan bool, MaxThreads)
	ctx, cancel := context.WithCancel(context.Background())
	rateLimiter := rate.NewLimiter(rate.Limit(mBytes*1000000), int(mBytes*1000000*2.0))
	chunkNumber := 0
	for {
		buf := make([]byte, ReadBufferSize)
		n, err := r.Read(buf)
		if err == io.EOF {
			chunkNumber = -1
		} else {
			chunkNumber++
		}
		if err := rateLimiter.WaitN(ctx, n); err != nil {
			cancel()
			log.Print(err)
			return
		}

		go func(b []byte, cNumber int) {
			sem <- true
			err = send(addr, b, cNumber, cancel)
			if err != nil {
				log.Print(err)
			}

			<-sem
		}(buf[:n], chunkNumber)

		if chunkNumber == -1 {
			break
		}
	}

WAIT:
	for {
		t := time.Tick(200 * time.Millisecond)
		select {
		case <-t:
			if len(sem) == 0 {
				break WAIT
			}
		}
	}
}

func send(addr string, buf []byte, c int, cancel context.CancelFunc) error {
	r := bytes.NewReader(buf)
	req, err := http.NewRequest("POST", "http://"+addr, r)
	if err != nil {
		cancel()
		return err
	}
	req.Header.Add("Content-type", "application/octet-stream")
	req.Header.Add(ChunkHeader, strconv.Itoa(c))

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		cancel()
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		cancel()
		return errors.New("Failed")
	}
	return nil
}
