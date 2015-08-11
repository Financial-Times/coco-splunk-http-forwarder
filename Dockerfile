FROM gliderlabs/alpine:3.2
ADD coco-splunk-http-forwarder /coco-splunk-http-forwarder
CMD coco-splunk-http-forwarder -url=$FORWARD_URL
