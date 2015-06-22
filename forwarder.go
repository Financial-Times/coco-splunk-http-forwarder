package main

import(
	"os"
	"bufio"
	"log"
	"io"
	"net/http"
	"strings"
	"crypto/tls"

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
	r, err := client.Post("https://user:pwd@splunk.glb.ft.com/coco-up/fleet", "application/json", strings.NewReader(s))
	if err != nil {
		log.Println(err)
	} else {
		if r.StatusCode != 200 {
			log.Printf("Unexpected status code %v" , r.StatusCode)
		}
	}

}

var client *http.Client
func init(){
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client = &http.Client{Transport: transport}
}
