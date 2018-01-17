package main

import (
	"flag"

	"bufio"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type s3ServiceMock struct {
	sync.RWMutex
	cache []string
}

var s3Mock = &s3ServiceMock{}

func (s3 *s3ServiceMock) Put(obj string) error {
	obj = strings.Replace(obj, "dispatch", "safe", -1)
	obj = strings.Replace(obj, "error", "dispatch", -1)
	s3.cache = append(s3.cache, obj)
	return nil
}

func TestMain(m *testing.M) {
	env = "dummy"
	workers = 8
	chanBuffer = 256
	batchsize = 10
	batchtimer = 5
	bucket = "testbucket"

	flag.Parse()

	NewS3Service = func(string, string, string) (S3Service, error) {
		return s3Mock, nil
	}

	os.Exit(m.Run())
}

func Test_Forwarder(t *testing.T) {
	in, out := io.Pipe()
	defer in.Close()

	br = bufio.NewReader(in)
	go main()
	messageCount := 100
	for i := 0; i < messageCount; i++ {
		out.Write([]byte(`127.0.0.1 - - [21/Apr/2015:12:15:34 +0000] "GET /eom-file/all/e09b49d6-e1fa-11e4-bb7f-00144feab7de HTTP/1.1" 200 53706 919 919` + "\n"))
	}
	time.Sleep(1 * time.Second)
	out.Close()
	assert.Equal(t, messageCount/batchsize, len(s3Mock.cache))
}
