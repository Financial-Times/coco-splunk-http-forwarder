package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
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
	batchtimer      int
	timerChan       = make(chan bool)
	timestampRegex  = regexp.MustCompile("([0-9]+)-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]|60)(.[0-9]+)?(([Zz])|([+|-]([01][0-9]|2[0-3]):[0-5][0-9]))")
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
	if len(hostname) == 0 { //Check whether -hostname parameter was provided. If not attempt to resolve
		hname, err := os.Hostname() //host name reported by the kernel, used for graphiteNamespace
		if err != nil {
			log.Println(err)
			hostname = "unkownhost" //Set host name as unkownhost if hostname resolution fail
		} else {
			hostname = hname
		}
	}

	log.Printf("Splunk forwarder (workers %v, buffer size %v, batchsize %v, batchtimer %v): Started\n", workers, chanBuffer, batchsize, batchtimer)
	defer log.Printf("Splunk forwarder: Stopped\n")
	logChan := make(chan string, chanBuffer)

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
	eventlist := make([]string, batchsize) //create eventlist slice that is size of -batchsize
	timerd := time.Duration(batchtimer) * time.Second
	timer := time.NewTimer(timerd) //create timer object with duration specified by -batchtimer
	go func() {                    //Create go routine for timer that writes into timerChan when it expires
		for {
			<-timer.C
			timerChan <- true
		}
	}()

	for {
		//1. Check whether timer has expired or batchsize exceeded before processing new string
		select { //set i equal to batchsize to trigger delivery if timer expires prior to batchsize limit is exceeded
		case <-timerChan:
			log.Println("Timer expired. Trigger delivery to Splunk")
			eventlist = stripEmptyStrings(eventlist) //remove empty values from slice before writing to channel
			i = batchsize
		default:
			break
		}
		if i >= batchsize { //Trigger delivery if batchsize is exceeded
			writeToLogChan(eventlist, logChan)
			i = 0 //reset i once batchsize is reached
			eventlist = nil
			eventlist = make([]string, batchsize)
			timer.Reset(timerd) //Reset timer after message delivery
		}
		//2. Process new string after ensuring eventlist has sufficient space
		str, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF { //Shutdown procedures: process eventlist, close channel and workers
				eventlist = stripEmptyStrings(eventlist) //remove empty values from slice before writing to channel
				if len(eventlist) > 0 {
					log.Printf("Processing %v batched messages before exit", len(eventlist))
					writeToLogChan(eventlist, logChan)
				}
				close(logChan)
				log.Printf("Waiting buffered channel consumer to finish processing messages\n")
				wg.Wait()
				return
			}
			log.Fatal(err)
		}

		//3. Append event on eventlist
		if i != batchsize {
			eventlist[i] = str
			i++
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

func stripEmptyStrings(eventlist []string) []string {
	//Find empty values in slice. Using map remove empties and return a slice without empty values
	i := 0
	map1 := make(map[int]string)
	for _, v := range eventlist {
		if v != "" {
			map1[i] = v
			i++
		}
	}
	mapToSlice := make([]string, len(map1))
	i = 0
	for _, v := range map1 {
		mapToSlice[i] = v
		i++
	}
	return mapToSlice
}

func writeJSON(eventlist []string) string {
	//Function produces Splunk HEC compatible json document for batched events
	// Example: { "event": "event 1"} { "event": "event 2"}
	var jsonDoc string

	for _, e := range eventlist {
		timestamp := timestampRegex.FindStringSubmatch(e)

		var err error
		var t = time.Now()
		if len(timestamp) > 0 {
			t, err = time.Parse(time.RFC3339, timestamp[0])
			if err != nil {
				t = time.Now()
			}
		}

		item := map[string]interface{}{"event": e, "time": t.Unix()}
		jsonItem, err := json.Marshal(&item)
		if err != nil {
			jsonDoc = strings.Join([]string{jsonDoc, strings.Join([]string{"{ \"event\":", e, "}"}, "")}, " ")
		} else {
			jsonDoc = strings.Join([]string{jsonDoc, string(jsonItem)}, " ")
		}
	}
	return jsonDoc
}

func writeToLogChan(eventlist []string, logChan chan string) {
	if len(eventlist) > 0 { //only attempt delivery if eventlist contains elements
		jsonSTRING := writeJSON(eventlist)
		t := metrics.GetOrRegisterTimer("post.queue.latency", metrics.DefaultRegistry)
		t.Time(func() {
			//log.Printf("Sending document to channel: %v", jsonSTRING)
			logChan <- jsonSTRING
		})
	}
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
	flag.IntVar(&batchtimer, "batchtimer", 5, "Expiry in seconds after which delivering events to Splunk HEC")

	flag.Parse()
}
