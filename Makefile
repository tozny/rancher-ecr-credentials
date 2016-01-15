build:
	GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build

deps:
	GO15VENDOREXPERIMENT=1 go get -u ./...

clean:
	rm rancher-ecr-credentials
