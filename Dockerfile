# syntax=docker/dockerfile:1

FROM golang:1.22 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile

COPY go.mod ./
COPY go.sum ./

RUN mkdir proto
COPY proto/*.go ./proto/

RUN mkdir server
COPY server/*.go ./server/

RUN mkdir db
COPY db/*.go ./db/

RUN mkdir queue_client
COPY queue_client/*.go ./queue_client


RUN mkdir background
COPY background/*.go ./background/

RUN mkdir config
COPY config/*.go ./config/

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /gramophile

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile /gramophile

EXPOSE 80
EXPOSE 8080
EXPOSE 8081
EXPOSE 8082
EXPOSE 8083

USER nonroot:nonroot

ENTRYPOINT ["/gramophile"]