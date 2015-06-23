from golang

RUN go get github.com/Financial-Times/coco-splunk-http-forwarder
CMD $GOPATH/bin/coco-splunk-http-forwarder -url=$FORWARD_URL