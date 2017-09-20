FROM golang:1.8.3-alpine

ENV PROJECT=coco-splunk-http-forwarder
COPY . /${PROJECT}-sources/

RUN apk --no-cache --virtual .build-dependencies add git \
  && ORG_PATH="github.com/Financial-Times" \
  && REPO_PATH="${ORG_PATH}/${PROJECT}" \
  && mkdir -p $GOPATH/src/${ORG_PATH} \
  && ln -s /${PROJECT}-sources $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && echo "Fetching dependencies..." \
  && go get . \
  && go build \
  && mv ${PROJECT} /${PROJECT} \
  && apk del .build-dependencies \
  && rm -rf $GOPATH /var/cache/apk/*

WORKDIR /

CMD exec /coco-splunk-http-forwarder -url=$FORWARD_URL -env=$ENV -hostname=$HOSTNAME -workers=$WORKERS -buffer=$BUFFER -token=$TOKEN -batchsize=$BATCHSIZE -batchtimer=$BATCHTIMER
