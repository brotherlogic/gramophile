# syntax=docker/dockerfile:1

FROM golang:1.25 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /gramophile-queue ./queue

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