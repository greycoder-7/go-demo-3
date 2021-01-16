FROM golang:1.15.3-alpine3.12 AS builder

RUN apk update \
     && apk add --no-cache git

ENV USER=gouser
ENV UID=1000    

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/empty" \
    --shell "/sbin/nologin" \    
    --no-create-home \
    --uid "$UID" \
    "$USER"

WORKDIR /go/src/github.com/jcostabe/go-demo/

COPY main.go .

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/src/github.com/jcostabe/go-demo/main

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder /go/src/github.com/jcostabe/go-demo/main /go/src/github.com/jcostabe/go-demo/main

USER gouser:gouser

ENTRYPOINT [ "/go/src/github.com/jcostabe/go-demo/main" ]  