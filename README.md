# Coco Splunk Http Forwarder

## Description
The Splunk forwarder is a golang application that posts a stdin to a provided URL.
Docker images builds a container that forwards the journalctl logs to the Splunk endpoint
 
## Usage
e.g. journalctl -f --output=json | $GOPATH/bin/coco-splunk-http-forwarder -url=$FORWARD_URL
