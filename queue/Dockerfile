# syntax=docker/dockerfile:1

FROM golang:1.23 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile-queue

COPY go.mod ./
COPY go.sum ./

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /gramophile-queue

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile-queue /gramophile-queue

USER nonroot:nonroot

EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["/gramophile-queue"]