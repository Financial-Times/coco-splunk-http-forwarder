package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	env            string
	workers        int
	chanBuffer     int
	batchsize      int
	batchtimer     int
	bucket         string
	awsRegion      string
	prefix         string
	br             *bufio.Reader
	timerChan      = make(chan bool)
	timestampRegex = regexp.MustCompile("([0-9]+)-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]|60)(.[0-9]+)?(([Zz])|([+|-]([01][0-9]|2[0-3]):[0-5][0-9]))")
	logDispatch    Dispatch
)

func main() {
	validateConfig()

	log.Printf("Splunk forwarder (workers %v, batchsize %v, batchtimer %v): Started\n", workers, batchsize, batchtimer)
	defer log.Printf("Splunk forwarder: Stopped\n")

	if br == nil {
		br = bufio.NewReader(os.Stdin)
	}
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

	logDispatch = NewDispatch(bucket, awsRegion, prefix)
	logDispatch.Start()

	for {
		//1. Check whether timer has expired or batchsize exceeded before processing new string
		select { //set i equal to batchsize to trigger delivery if timer expires prior to batchsize limit is exceeded
		case <-timerChan:
			log.Println("Timer expired. Trigger delivery to S3")
			eventlist = stripEmptyStrings(eventlist) //remove empty values from slice before writing to channel
			i = batchsize
		default:
			break
		}
		if i >= batchsize { //Trigger delivery if batchsize is exceeded
			processAndEnqueue(eventlist)
			i = 0 //reset i once batchsize is reached
			eventlist = nil
			eventlist = make([]string, batchsize)
			timer.Reset(timerd) //Reset timer after message delivery
		}
		//2. Process new string after ensuring eventlist has sufficient space
		str, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF { //Shutdown procedures: process eventlist, close workers
				eventlist = stripEmptyStrings(eventlist) //remove empty values from slice before writing to channel
				if len(eventlist) > 0 {
					log.Printf("Processing %v batched messages before exit", len(eventlist))
					processAndEnqueue(eventlist)
				}
				logDispatch.Stop()
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

func validateConfig() {
	if len(bucket) == 0 { //Check whether -bucket parameter value was provided
		log.Printf("-bucket=bucket_name\n")
		os.Exit(1) //If not fail visibly as we are unable to send logs to Splunk
	}
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
			t, err = time.Parse(time.RFC3339Nano, timestamp[0])
			if err != nil {
				t = time.Now()
			}
		}

		// For Splunk HEC, the default time format is epoch time format, in the format <sec>.<ms>.
		// For example, 1433188255.500 indicates 1433188255 seconds and 500 milliseconds after epoch, or Monday, June 1, 2015, at 7:50:55 PM GMT.
		epochMillis, err := strconv.ParseFloat(fmt.Sprintf("%d.%03d", t.Unix(), t.Nanosecond()/int(time.Millisecond)), 64)
		if err != nil {
			epochMillis = float64(t.UnixNano()) / float64(time.Second)
		}
		item := map[string]interface{}{"event": e, "time": epochMillis}
		jsonItem, err := json.Marshal(&item)
		if err != nil {
			jsonDoc = strings.Join([]string{jsonDoc, strings.Join([]string{"{ \"event\":", e, "}"}, "")}, " ")
		} else {
			jsonDoc = strings.Join([]string{jsonDoc, string(jsonItem)}, " ")
		}
	}
	return jsonDoc
}

func processAndEnqueue(eventlist []string) {
	if len(eventlist) > 0 { //only attempt delivery if eventlist contains elements
		jsonSTRING := writeJSON(eventlist)
		logDispatch.Enqueue(jsonSTRING)
	}
}

func init() {
	flag.StringVar(&env, "env", "dummy", "environment_tag value")
	flag.IntVar(&workers, "workers", 8, "Number of concurrent workers")
	flag.IntVar(&chanBuffer, "buffer", 256, "Channel buffer size")
	flag.IntVar(&batchsize, "batchsize", 10, "Number of messages to group before delivering to Splunk HEC")
	flag.IntVar(&batchtimer, "batchtimer", 5, "Expiry in seconds after which delivering events to Splunk HEC")
	flag.StringVar(&bucket, "bucketName", "", "S3 bucket for caching failed events")
	flag.StringVar(&awsRegion, "awsRegion", "", "AWS region for S3")
	flag.StringVar(&prefix, "prefix", "global", "S3 id prefix for this instance")

	flag.Parse()
}
