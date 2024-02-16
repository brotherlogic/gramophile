# syntax=docker/dockerfile:1

FROM golang:1.20 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile-validator

COPY go.mod ./
COPY go.sum ./

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /gramophile-validator

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile-validator /gramophile-validator

USER nonroot:nonroot

ENTRYPOINT ["/gramophile-validator"]