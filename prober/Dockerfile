# syntax=docker/dockerfile:1

FROM golang:1.25 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/
COPY . .

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/prober
RUN go mod download
RUN CGO_ENABLED=0 go build -o /gramophile-prober

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /gramophile-prober /gramophile-prober

USER nonroot:nonroot


ENTRYPOINT ["/gramophile-prober"]