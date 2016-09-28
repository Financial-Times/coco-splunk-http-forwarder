package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const workers = 8

func main() {
	log.Println("Splunk forwarder: Started")
	defer log.Println("Splunk forwarder: Stopped")

	logChan := make(chan string)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range logChan {
				postToSplunk(msg)					
			}
		}()
	}

	br := bufio.NewReader(os.Stdin)
	for {
		str, err := br.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				close(logChan)
				return
			}
			log.Fatal(err)
		}
		logChan <- str
	}

	wg.Wait()
}

func postToSplunk(s string) {
	r, err := client.Post(fwdUrl, "application/json", strings.NewReader(s))
	if err != nil {
		log.Println(err)
	} else {
		defer r.Body.Close()
		io.Copy(ioutil.Discard, r.Body)
		if r.StatusCode != 200 {
			log.Printf("Unexpected status code %v when sending %v to %v", r.StatusCode, s, fwdUrl)
		}
	}

}

var client *http.Client
var fwdUrl string

func init() {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: workers,
	}
	client = &http.Client{Transport: transport}

	flag.StringVar(&fwdUrl, "url", "https://user:pwd@splunk.glb.ft.com/coco-up/fleet", "The url to forward to")
	flag.Parse()
}
