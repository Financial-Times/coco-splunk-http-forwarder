package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pborman/uuid"
)

type S3Service interface {
	Put(obj string) error
}

type s3Service struct {
	bucketName string
	prefix     string
	svc        *s3.S3
}

var NewS3Service = func(bucketName string, awsRegion string, prefix string) (S3Service, error) {
	wrks := workers
	spareWorkers := 1

	hc := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          wrks + spareWorkers,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   wrks + spareWorkers,
			TLSHandshakeTimeout:   3 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	sess, err := session.NewSession(
		&aws.Config{
			Region:     aws.String(awsRegion),
			MaxRetries: aws.Int(1),
			HTTPClient: hc,
		})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
		return nil, err
	}
	svc := s3.New(sess)
	return &s3Service{bucketName, prefix, svc}, nil
}

func (s *s3Service) Put(obj string) error {
	uuid := fmt.Sprintf("%v/%v_%v", s.prefix, string(time.Now().UnixNano()), uuid.New())
	_, err := s.svc.PutObject(&s3.PutObjectInput{
		Bucket: &s.bucketName,
		Body:   strings.NewReader(obj),
		Key:    &uuid})
	return err
}
