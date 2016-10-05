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
	
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/rcrowley/go-metrics"

)

var (
    client *http.Client
    fwdUrl string
    env string
    dryrun bool
    workers int
    graphitePrefix string = "coco.services"
    graphitePostfix string = "splunk-forwarder"
    graphiteServer string
    counter = 0
    )


func main() {
    if len(fwdUrl) == 0 { //Check whether -url parameter was provided 
        log.Printf("-url=http_endpoint parameter must be provided")
        os.Exit(1) //If not fail visibly as we are unable to send logs to Splunk
    }
    
	log.Printf("Splunk forwarder (%v workers): Started", workers)
	defer log.Println("Splunk forwarder: Stopped")
    //logChan := make(chan string, 256)
    logChan := make(chan string)

    hostname, err := os.Hostname() //host name reported by the kernel, used for graphiteNamespace
    if err != nil {
        log.Println(err)
    }
    graphiteNamespace := strings.Join([]string{graphitePrefix, env, graphitePostfix, hostname}, ".") // Join prefix, env and postfix
    log.Printf("%v namespace: %v", graphiteServer, graphiteNamespace)
    if dryrun {
        log.Printf("Dryrun enabled, not connecting to %v", graphiteServer)
    } else {
        addr, err := net.ResolveTCPAddr("tcp", graphiteServer)
        if err != nil {
            log.Println(err)
        }
        go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, graphitePrefix, addr)        
    }
    go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.New(os.Stdout, "metrics ", log.Lmicroseconds))
    
	go queueLenMetrics(logChan)

	for i := 0; i < workers; i++ {
		//log.Printf("Starting worker %v", i)
		go func() {
			for msg := range logChan {
                if dryrun {
                    log.Printf("Dryrun enabled, not posting to %v", fwdUrl)
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
				return
			}
			log.Fatal(err)
		} 
		t := metrics.GetOrRegisterTimer("post.queue.latency", metrics.DefaultRegistry)
		t.Time(func() {
		  counter++
		  log.Printf("Delivering event %v to logChan", counter)
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
        log.Printf("Posting event %v to splunk", counter)
        r, err := client.Post(fwdUrl, "application/json", strings.NewReader(s))
        if err != nil {
            log.Println(err)
        } else {
            log.Printf("Processing HTTP response code %v", r.StatusCode)
            defer r.Body.Close()
            io.Copy(ioutil.Discard, r.Body)
            if r.StatusCode != 200 {
                log.Printf("Unexpected status code %v when sending %v to %v", r.StatusCode, s, fwdUrl)
            } else {
                log.Printf("Successfully (%v) sent to endpoint %v", r.StatusCode, fwdUrl)
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
	flag.Parse()
}
