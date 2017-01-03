build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 govendor build +local

clean:
	rm rancher-ecr-credentials
