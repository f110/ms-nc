package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	ChunkHeader = "chunk-number"
)

type MultiStreamReaderWriter struct {
	CurrentChunk int
	Writer       io.Writer
	Buf          map[int][]byte
	mutex        sync.RWMutex
	Ctx          context.Context
	Cancel       context.CancelFunc
}

func NewMultiStreamReaderWriter(w io.Writer) *MultiStreamReaderWriter {
	ctx, cancel := context.WithCancel(context.Background())
	rw := &MultiStreamReaderWriter{
		CurrentChunk: 0,
		Writer:       w,
		Ctx:          ctx,
		Cancel:       cancel,
		Buf:          make(map[int][]byte),
	}

	go func() {
		t := time.Tick(200 * time.Millisecond)
	WRITE:
		for {
			select {
			case <-t:
				rw.mutex.RLock()
				if buf, ok := rw.Buf[rw.CurrentChunk+1]; ok {
					n, err := rw.Writer.Write(buf)
					if len(buf) != n || err != nil {
						rw.Cancel()
					}
					delete(rw.Buf, rw.CurrentChunk+1)
					rw.CurrentChunk++
				}
				rw.mutex.RUnlock()
			case <-rw.Ctx.Done():
				break WRITE
			}
		}
	}()

	return rw
}

func (rw *MultiStreamReaderWriter) Write(r io.ReadCloser, c int) error {
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	rw.mutex.Lock()
	rw.Buf[c] = buf
	rw.mutex.Unlock()

	return nil
}

func (rw *MultiStreamReaderWriter) Finish(r io.ReadCloser) error {
	defer r.Close()

	return nil
}

func NewListener(w io.Writer) {
	rw := NewMultiStreamReaderWriter(w)

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
	log.Fatal(http.ListenAndServe("127.0.0.1:30000", nil))
}
