package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"net/http/httptest"

	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type splunkMock struct {
	sync.RWMutex
	index      []string
	errorCount int
}

func (splunk *splunkMock) append(s string) {
	splunk.Lock()
	defer splunk.Unlock()
	splunk.index = append(splunk.index, s)
}

func (splunk *splunkMock) getIndex() []string {
	splunk.Lock()
	defer splunk.Unlock()
	return splunk.index
}

func (splunk *splunkMock) incErrors() {
	splunk.Lock()
	defer splunk.Unlock()
	splunk.errorCount++
}

func (splunk *splunkMock) getErrorCount() int {
	splunk.Lock()
	defer splunk.Unlock()
	return splunk.errorCount
}

type s3ServiceMock struct {
	sync.RWMutex
	cache []string
}

var splunk = splunkMock{}

func (s3 *s3ServiceMock) ListAndDelete() ([]string, error) {
	items := s3.cache
	s3.cache = make([]string, 0)
	return items, nil
}

func (s3 *s3ServiceMock) Put(obj string) error {
	obj = strings.Replace(obj, "error", "retry", -1)
	s3.cache = append(s3.cache, obj)
	return nil
}

func TestMain(m *testing.M) {

	splunkTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes := make([]byte, r.ContentLength)
		r.Body.Read(bytes)
		defer r.Body.Close()
		body := string(bytes)
		if strings.Contains(body, "simulated_error") {
			splunk.incErrors()
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			splunk.append(body)
			w.WriteHeader(http.StatusOK)
		}
	}))

	defer splunkTestServer.Close()

	graphiteTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	defer graphiteTestServer.Close()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: workers,
	}
	client = &http.Client{Transport: transport}

	fwdURL = splunkTestServer.URL
	env = "dummy"
	graphiteURL, _ := url.Parse(graphiteTestServer.URL)
	graphiteServer = fmt.Sprintf("%v:%v", graphiteURL.Hostname(), graphiteURL.Port())
	dryrun = false
	workers = 8
	chanBuffer = 256
	hostname = ""
	token = "secret"
	batchsize = 10
	batchtimer = 5
	bucket = "testbucket"

	flag.Parse()

	NewS3Service = func(string, string) (S3Service, error) {
		return &s3ServiceMock{}, nil
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
		if i == 50 {
			out.Write([]byte("simulated_error\n"))
		} else {
			out.Write([]byte(`127.0.0.1 - - [21/Apr/2015:12:15:34 +0000] "GET /eom-file/all/e09b49d6-e1fa-11e4-bb7f-00144feab7de HTTP/1.1" 200 53706 919 919` + "\n"))
		}
	}
	out.Close()
	time.Sleep(3 * time.Second)
	assert.Equal(t, messageCount/batchsize, len(splunk.getIndex()))
	assert.Equal(t, 1, splunk.getErrorCount())
	assert.Contains(t, strings.Join(splunk.getIndex(), ""), "simulated_retry")
}
