package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	ChunkHeader   = "chunk-number"
	MaxBufferSize = 512000000 // 512MB
)

type MultiStreamWriter struct {
	CurrentChunk    int
	Writer          io.Writer
	Listener        net.Listener
	Buf             map[int][]byte
	LastBuf         []byte
	TransportFinish bool
	mutex           sync.RWMutex
	Ctx             context.Context
	Cancel          context.CancelFunc
	BufferSize      int
}

func NewMultiStreamWriter(w io.Writer, l net.Listener) *MultiStreamWriter {
	ctx, cancel := context.WithCancel(context.Background())
	rw := &MultiStreamWriter{
		CurrentChunk:    0,
		Writer:          w,
		Listener:        l,
		TransportFinish: false,
		Ctx:             ctx,
		Cancel:          cancel,
		Buf:             make(map[int][]byte),
		BufferSize:      0,
	}

	go func() {
		t := time.Tick(200 * time.Millisecond)
	WRITE:
		for {
			select {
			case <-t:
				rw.mutex.Lock()
				if buf, ok := rw.Buf[rw.CurrentChunk+1]; ok {
					n, err := rw.Writer.Write(buf[:])
					if len(buf) != n || err != nil {
						rw.Cancel()
					}
					delete(rw.Buf, rw.CurrentChunk+1)
					rw.BufferSize -= len(buf)
					rw.CurrentChunk++
				}
				if rw.TransportFinish && len(rw.Buf) == 0 {
					rw.Writer.Write(rw.LastBuf[:])
					break WRITE
				}
				rw.mutex.Unlock()
			case <-rw.Ctx.Done():
				break WRITE
			}
		}

		rw.Listener.Close()
	}()

	return rw
}

func (rw *MultiStreamWriter) Write(r io.ReadCloser, c int) error {
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	rw.mutex.Lock()
	if rw.BufferSize+len(buf) > MaxBufferSize {
		rw.Cancel()
	}
	rw.Buf[c] = buf[:]
	rw.BufferSize += len(buf)
	rw.mutex.Unlock()

	return nil
}

func (rw *MultiStreamWriter) Finish(r io.ReadCloser) error {
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	rw.mutex.Lock()
	rw.LastBuf = buf[:]
	rw.TransportFinish = true
	rw.mutex.Unlock()

	return nil
}

func NewReceiver(port int, w io.Writer) {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Print(err)
		return
	}
	rw := NewMultiStreamWriter(w, l)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get(ChunkHeader) == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		c, err := strconv.Atoi(r.Header.Get(ChunkHeader))
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
		}

		if c == -1 {
			err := rw.Finish(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			err := rw.Write(r.Body, c)
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	})
	http.Serve(l, nil)
}
