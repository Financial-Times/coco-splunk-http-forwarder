package main

import (
	"log"
	"math"
	"time"
)

const (
	sleepTime  = 100
	maxBackoff = 9
	minBackoff = 2
)

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
	action        func(string) error
	statusChecker func() serviceStatus
	cache         S3Service
}

func NewRetry(action func(string) error, statusChecker func() serviceStatus, bucketName string, awsRegion string) Retry {
	svc, _ := NewS3Service(bucketName, awsRegion)
	return retry{action, statusChecker, svc}
}

func (logRetry retry) Start() {
	go func() {
		level := 3
		for {
			status := logRetry.statusChecker()
			if status.healthy {
				entries, err := logRetry.Dequeue()
				if err != nil {
					log.Printf("Failure retrieving logs from S3 %v\n", err)
				} else {
					log.Printf("Read %v messages from S3\n", len(entries))
				}
				for _, entry := range entries {
					log.Printf("Retrying for message %v\n", entry)
					err := logRetry.action(entry)
					if err != nil {
						if level < maxBackoff {
							level++
						}
					} else {
						if level > minBackoff {
							level--
						}
					}
					sleepDuration := time.Duration((0.15*math.Pow(2, float64(level))-0.2)*1000) * time.Millisecond
					log.Printf("Sleeping for %v\n", sleepDuration)
					time.Sleep(sleepDuration)
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
