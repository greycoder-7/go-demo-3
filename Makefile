VERSION := 1.0
REPO := jcostabe
NAME := go-demo-3
MAIN := main.go
CURRENT_PATH := .
EXTERNAL_PORT := "8080"
INTERNAL_PORT := "8080"


buid_go:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/src/github.com/jcostabe/go-demo/main

run_go:
	go run ${MAIN}

build:
	docker build -t ${REPO}\/${NAME}\:${VERSION} ${CURRENT_PATH}

push:
	docker push ${REPO}\/${NAME}\:${VERSION}

run:
	docker run -d --name ${NAME} -p ${EXTERNAL_PORT}:${INTERNAL_PORT} ${REPO}\/${NAME}\:${VERSION}

delete:
	docker rm -f ${NAME}

clean:
	rm -f main

DEFAULT: build
