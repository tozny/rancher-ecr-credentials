FROM golang:alpine

RUN apk add --no-cache git \
    && go get github.com/kardianos/govendor
RUN mkdir -p /go/src/rancher-ecr-credentials
COPY . /go/src/rancher-ecr-credentials/
WORKDIR /go/src/rancher-ecr-credentials/
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 govendor build -a -installsuffix cgo \
  && mv rancher-ecr-credentials /bin/ \
  && rm -rf /go/src/rancher-ecr-credentials

ENTRYPOINT ["/bin/rancher-ecr-credentials"]
