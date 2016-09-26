package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"
)

const workers = 8

var (
	client *http.Client
	fwdUrl string

	//graphitePrefix string             = "coco.services.$ENV.splunk-forwarder-$MACHINE"
	graphitePrefix string = "coco.services.dummy.splunk-forwarder-foo" // FIXME: don't hardcode
)

func main() {

	addrStr := "graphite.ft.com:2003" // FIXME: don't hardcode
	addr, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		panic(err)
	}

	go graphite.Graphite(metrics.DefaultRegistry, 2*time.Second, graphitePrefix, addr)
	go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))

	log.Println("Splunk forwarder: Started")
	defer log.Println("Splunk forwarder: Stopped")

	forSplunk := make(chan string, 256)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for msg := range forSplunk {
				postToSplunk(msg)
			}
		}()
	}

	br := bufio.NewReader(os.Stdin)
	for {
		str, err := br.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				close(forSplunk)
				return
			}
			log.Fatal(err)
		}
		t := metrics.GetOrRegisterTimer("post.queue.latency", nil)
		t.Time(func() {
			forSplunk <- str
		})

	}

	wg.Wait()
}

func postToSplunk(s string) {
	t := metrics.GetOrRegisterTimer("post.time", nil)
	t.Time(func() {
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
	})
}

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
