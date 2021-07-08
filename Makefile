build:
	go build -v

update-module:
	GO111MODULE=on go get -u github.com/hhatto/ftcat

cleanup-module:
	GO111MODULE=on go mod tidy

lint:
	golint *.go
