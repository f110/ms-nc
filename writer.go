package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
)

const (
	ReadBufferSize = 5120000
)

func NewWriter(addr string, r io.Reader) {
	buf := make([]byte, ReadBufferSize)
	chunkNumber := 1
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			chunkNumber = -1
		}
		if n == ReadBufferSize {
			chunkNumber++
		}

		err = send(addr, buf[:n], chunkNumber)
		if err != nil {
			log.Print(err)
		}
		if chunkNumber == -1 {
			return
		}
	}
}

func send(addr string, buf []byte, c int) error {
	r := bytes.NewReader(buf)
	req, err := http.NewRequest("POST", "http://"+addr, r)
	if err != nil {
		return err
	}
	req.Header.Add("Content-type", "application/octet-stream")
	req.Header.Add(ChunkHeader, strconv.Itoa(c))

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("Failed")
	}
	return nil
}
