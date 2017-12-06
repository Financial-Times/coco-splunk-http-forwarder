package main

import (
	"log"
	"time"
)

const sleepTime = 100

type Retry interface {
	Start()
	Enqueue(s string) error
	Dequeue() ([]string, error)
}

type serviceStatus struct {
	healthy   bool
	timestamp time.Time
}

type retry struct {
	action        func(string)
	statusChecker func() serviceStatus
	cache         S3Service
}

func NewRetry(action func(string), statusChecker func() serviceStatus, bucketName string, awsRegion string) Retry {
	svc, _ := NewS3Service(bucketName, awsRegion)
	return retry{action, statusChecker, svc}
}

func (logRetry retry) Start() {
	go func() {
		for {
			status := logRetry.statusChecker()
			if status.healthy {
				entries, err := logRetry.Dequeue()
				if err != nil {
					log.Printf("Failure retrieving logs from S3 %v\n", err)
				}
				for _, entry := range entries {
					log.Printf("Retrying for message %v", entry)
					logRetry.action(entry)
				}
			}
			time.Sleep(sleepTime * time.Millisecond)
		}
	}()
}

func (logRetry retry) Enqueue(s string) error {
	return logRetry.cache.Put(s)
}

func (logRetry retry) Dequeue() ([]string, error) {
	return logRetry.cache.ListAndDelete()
}
