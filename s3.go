package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pborman/uuid"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

const maxKeys = int64(100)

type S3Service interface {
	ListAndDelete() ([]string, error)
	Put(obj string) error
}

type s3Service struct {
	bucketName string
	svc        *s3.S3
}

var NewS3Service = func(bucketName string, awsRegion string) (S3Service, error) {
	wrks := 2
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
	return &s3Service{bucketName, svc}, nil
}

func (s *s3Service) ListAndDelete() ([]string, error) {
	mK := maxKeys
	out, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  &s.bucketName,
		MaxKeys: &mK,
	})
	if err != nil {
		return nil, err
	}
	ids := []*s3.ObjectIdentifier{}
	vals := []string{}
	for _, obj := range out.Contents {
		ids = append(ids, &s3.ObjectIdentifier{Key: obj.Key})

		val, err := s.Get(*obj.Key)
		if err != nil {
			return nil, err
		}

		vals = append(vals, val)
	}

	_, err = s.svc.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: &s.bucketName,
		Delete: &s3.Delete{
			Objects: ids,
		},
	})
	if err != nil {
		return nil, err
	}
	return vals, nil
}

func (s *s3Service) Put(obj string) error {
	uuid := uuid.New()
	_, err := s.svc.PutObject(&s3.PutObjectInput{
		Bucket: &s.bucketName,
		Body:   strings.NewReader(obj),
		Key:    &uuid})
	return err
}

func (s *s3Service) Get(key string) (string, error) {
	val, err := s.svc.GetObject(&s3.GetObjectInput{
		Bucket: &s.bucketName,
		Key:    &key,
	})
	if err != nil {
		return "", err
	}

	defer val.Body.Close()
	buf := make([]byte, *val.ContentLength)
	val.Body.Read(buf)
	return string(buf), nil
}
