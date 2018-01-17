# Coco Splunk Http Forwarder

## Building
```
CGO_ENABLED=0 go build -a -installsuffix cgo -o coco-splunk-http-forwarder .

docker build -t coco/coco-splunk-http-forwarder .
```

## Description
The Splunk forwarder is a golang application that posts a stdin to S3 in order to be processed by the resilient Splunk forwarder.
Docker image builds a container that stores the journalctl logs to S3.
 
## Usage ex
e.g. journalctl -f --output=json | ./coco-splunk-http-forwarder -env=$ENV -workers=$WORKERS -buffer=$BUFFER -batchsize=$BATCHSIZE -batchtimer=$BATCHTIMER -bucketName=$BUCKET_NAME -prefix=$PREFIX -awsRegion=$AWS_REGION
