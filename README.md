# Coco Splunk Http Forwarder

## Building
```
CGO_ENABLED=0 go build -a -installsuffix cgo -o coco-splunk-http-forwarder .

docker build -t coco/coco-splunk-http-forwarder .
```

## Description
The Splunk forwarder is a golang application that posts a stdin to a provided URL.
Docker images builds a container that forwards the journalctl logs to the Splunk endpoint
 
## Usage ex
e.g. journalctl -f --output=json | ./coco-splunk-http-forwarder -url=$FORWARD_URL
