# syntax=docker/dockerfile:1

FROM golang:1.22 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/overseer

COPY go.mod ./
COPY go.sum ./

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /overseer

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile-queue /overseer

USER nonroot:nonroot

EXPOSE 80

ENTRYPOINT ["/overseer"]