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
	"time"
	"sync"

	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"

)

var (
    wg sync.WaitGroup
    client *http.Client
    fwdUrl string
    env string
    dryrun bool
    workers int
    graphitePrefix string = "coco.services"
    graphitePostfix string = "splunk-forwarder"
    graphiteServer string
    chan_buffer int
		hostname string
    )


func main() {
    if len(fwdUrl) == 0 { //Check whether -url parameter was provided
        log.Printf("-url=http_endpoint parameter must be provided\n")
        os.Exit(1) //If not fail visibly as we are unable to send logs to Splunk
    }

		log.Printf("Splunk forwarder (workers %v, buffer size %v): Started\n", workers, chan_buffer)
		defer log.Printf("Splunk forwarder: Stopped\n")
    logChan := make(chan string, chan_buffer)

		if len(hostname) == 0 { //Check whether -hostname parameter was provided. If not attempt to resolve
    	hname, err := os.Hostname() //host name reported by the kernel, used for graphiteNamespace
    	if err != nil {
      	log.Println(err)
				hostname="unkownhost" //Set host name as unkownhost if hostname resolution fail
    	} else {
				hostname = hname
			}
		}

    graphiteNamespace := strings.Join([]string{graphitePrefix, env, graphitePostfix, hostname}, ".") // graphiteNamespace ~ prefix.env.postfix.hostname
    log.Printf("%v namespace: %v\n", graphiteServer, graphiteNamespace)
    if dryrun {
        log.Printf("Dryrun enabled, not connecting to %v\n", graphiteServer)
    } else {
        addr, err := net.ResolveTCPAddr("tcp", graphiteServer)
        if err != nil {
            log.Println(err)
        }
    go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, graphiteNamespace, addr)
    }
    go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stdout, "metrics ", log.Lmicroseconds))

		go queueLenMetrics(logChan)

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
      	defer wg.Done()
				for msg := range logChan {
					if dryrun {
          	log.Printf("Dryrun enabled, not posting to %v\n", fwdUrl)
					} else {
						postToSplunk(msg)
          }
			}
		}()
	}

	br := bufio.NewReader(os.Stdin)
	for {
		str, err := br.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				close(logChan)
				log.Printf("Waiting buffered channel consumer to finish processing messages\n")
				wg.Wait()
				return
			}
			log.Fatal(err)
		}
		t := metrics.GetOrRegisterTimer("post.queue.latency", metrics.DefaultRegistry)
		t.Time(func() {
		  logChan <- str
		})
	}
}

func queueLenMetrics(queue chan string) {
    s := metrics.NewExpDecaySample(1024, 0.015)
    h := metrics.GetOrRegisterHistogram("post.queue.length", metrics.DefaultRegistry, s)
    for {
        time.Sleep(200 * time.Millisecond)
        h.Update(int64(len(queue)))
    }
}

func postToSplunk(s string) {
    t := metrics.GetOrRegisterTimer("post.time", metrics.DefaultRegistry)
    t.Time(func() {
        r, err := client.Post(fwdUrl, "application/json", strings.NewReader(s))
        if err != nil {
            log.Println(err)
        } else {
            defer r.Body.Close()
            io.Copy(ioutil.Discard, r.Body)
            if r.StatusCode != 200 {
                log.Printf("Unexpected status code %v when sending %v to %v\n", r.StatusCode, s, fwdUrl)
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

	flag.StringVar(&fwdUrl, "url", "", "The url to forward to")
	flag.StringVar(&env, "env", "dummy", "environment_tag value")
	flag.StringVar(&graphiteServer, "graphiteserver", "graphite.ft.com:2003", "Graphite server host name and port")
	flag.BoolVar(&dryrun, "dryrun", false, "Dryrun true disables network connectivity. Use it for testing offline. Default value false")
	flag.IntVar(&workers, "workers", 8, "Number of concurrent workers")
	flag.IntVar(&chan_buffer, "buffer", 256, "Channel buffer size")
	flag.StringVar(&hostname, "hostname", "", "Hostname running the service. If empty Go is trying to resolve the hostname.")
	flag.Parse()
}
