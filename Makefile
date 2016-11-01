build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build

deps:
	go get -t -d -v ./...

clean:
	rm rancher-ecr-credentials
