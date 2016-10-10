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

var (
	wg              sync.WaitGroup
	client          *http.Client
	fwdURL          string
	env             string
	dryrun          bool
	workers         int
	graphitePrefix  = "coco.services"
	graphitePostfix = "splunk-forwarder"
	graphiteServer  string
	chanBuffer      int
	hostname        string
	token           string
	batchsize       int
)

func main() {
	if len(fwdURL) == 0 { //Check whether -url parameter value was provided
		log.Printf("-url=http_endpoint parameter must be provided\n")
		os.Exit(1) //If not fail visibly as we are unable to send logs to Splunk
	}
	if len(token) == 0 { //Check whether -token parameter value was provided
		log.Printf("-token=secret must be provided\n")
		os.Exit(1) //If not fail visibly as we are unable to send logs to Splunk
	}

	log.Printf("Splunk forwarder (workers %v, buffer size %v): Started\n", workers, chanBuffer)
	defer log.Printf("Splunk forwarder: Stopped\n")
	logChan := make(chan string, chanBuffer)

	if len(hostname) == 0 { //Check whether -hostname parameter was provided. If not attempt to resolve
		hname, err := os.Hostname() //host name reported by the kernel, used for graphiteNamespace
		if err != nil {
			log.Println(err)
			hostname = "unkownhost" //Set host name as unkownhost if hostname resolution fail
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
					log.Printf("Dryrun enabled, not posting to %v\n", fwdURL)
				} else {
					postToSplunk(msg)
				}
			}
		}()
	}

	br := bufio.NewReader(os.Stdin)
	i := 0
	eventlist := make([]string, batchsize) //create slice size of batchsize
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
		if i >= batchsize {
			jsonSTRING := writeJSON(eventlist)
			t := metrics.GetOrRegisterTimer("post.queue.latency", metrics.DefaultRegistry)
			t.Time(func() {
				logChan <- jsonSTRING
			})
			i = 0 //reset i once batchsize is reached
		} else {
			eventlist[i] = str
			i++ //increment i
		}
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
		req, err := http.NewRequest("POST", fwdURL, strings.NewReader(s))
		if err != nil {
			log.Println(err)
		}
		tokenWithKeyword := strings.Join([]string{"Splunk", token}, " ") //join strings "Splunk" and value of -token argument
		req.Header.Set("Authorization", tokenWithKeyword)
		r, err := client.Do(req)
		if err != nil {
			log.Println(err)
		} else {
			defer r.Body.Close()
			io.Copy(ioutil.Discard, r.Body)
			if r.StatusCode != 200 {
				log.Printf("Unexpected status code %v (%v) when sending %v to %v\n", r.StatusCode, r.Status, s, fwdURL)
			}
		}
	})
}
func writeJSON(eventlist []string) string {
	jsonPREFIX := "{ \"event\":"
	jsonPOSTFIX := "}"
	jsonDOC := strings.Join(eventlist, "} { \"event\":")
	jsonDOC = strings.Join([]string{jsonPREFIX, jsonDOC, jsonPOSTFIX}, " ")
	return jsonDOC
}

func init() {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConnsPerHost: workers,
	}
	client = &http.Client{Transport: transport}

	flag.StringVar(&fwdURL, "url", "", "The url to forward to")
	flag.StringVar(&env, "env", "dummy", "environment_tag value")
	flag.StringVar(&graphiteServer, "graphiteserver", "graphite.ft.com:2003", "Graphite server host name and port")
	flag.BoolVar(&dryrun, "dryrun", false, "Dryrun true disables network connectivity. Use it for testing offline. Default value false")
	flag.IntVar(&workers, "workers", 8, "Number of concurrent workers")
	flag.IntVar(&chanBuffer, "buffer", 256, "Channel buffer size")
	flag.StringVar(&hostname, "hostname", "", "Hostname running the service. If empty Go is trying to resolve the hostname.")
	flag.StringVar(&token, "token", "", "Splunk HEC Authorization token")
	flag.IntVar(&batchsize, "batchsize", 10, "Number of messages to group before delivering to Splunk HEC")
	flag.Parse()
}
