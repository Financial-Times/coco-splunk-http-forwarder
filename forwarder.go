package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"fmt"
	"flag"
)

func main() {
	br := bufio.NewReader(os.Stdin)
	for {
		str, err := br.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}
		postToSplunk(str)
	}
}

func postToSplunk(s string) {
	r, err := client.Post(fwdUrl, "application/json", strings.NewReader(s))
	if err != nil {
		log.Println(err)
	} else {
		if r.StatusCode != 200 {
			log.Printf("Unexpected status code %v when sending %v to %v", r.StatusCode, s, fwdUrl)
		}
	}

}

var client *http.Client
var fwdUrl string
func init() {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client = &http.Client{Transport: transport}

	flag.StringVar(&fwdUrl, "url", "https://user:pwd@splunk.glb.ft.com/coco-up/fleet", "The url to forward to")
	flag.Parse()
}
