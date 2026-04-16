# syntax=docker/dockerfile:1

FROM golang:1.26.2 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/
COPY . .

WORKDIR $GOPATH/src/github.com/brotherlogic/gramophile/overseer
RUN go mod download
RUN CGO_ENABLED=0 go build -o /overseer

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /overseer /overseer

USER nonroot:nonroot

EXPOSE 80

ENTRYPOINT ["/overseer"]