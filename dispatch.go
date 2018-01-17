package main

import (
	"log"
	"sync"
)

type Dispatch interface {
	Start()
	Stop()
	Enqueue(s string)
}

type dispatch struct {
	cache  S3Service
	inChan chan string
	wg     sync.WaitGroup
}

func NewDispatch(bucketName string, awsRegion string, prefix string) Dispatch {
	svc, _ := NewS3Service(bucketName, awsRegion, prefix)
	return &dispatch{svc, nil, sync.WaitGroup{}}
}

func (logDispatch *dispatch) Start() {
	logDispatch.inChan = make(chan string, chanBuffer)
	for i := 0; i < workers; i++ {
		logDispatch.wg.Add(1)
		go func() {
			defer logDispatch.wg.Done()
			for msg := range logDispatch.inChan {
				err := logDispatch.cache.Put(msg)
				if err != nil {
					log.Printf("Unexpected error when caching messages: %v\n", err)
				}
			}
		}()
	}
}

func (logDispatch *dispatch) Stop() {
	close(logDispatch.inChan)
	log.Printf("Waiting buffered channel consumer to finish processing messages\n")
	logDispatch.wg.Wait()
}

func (logDispatch *dispatch) Enqueue(s string) {
	logDispatch.inChan <- s
}
