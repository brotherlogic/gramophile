# syntax=docker/dockerfile:1

FROM golang:1.26 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /gramophile-validator ./validator

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile-validator /gramophile-validator

USER nonroot:nonroot

ENTRYPOINT ["/gramophile-validator"]