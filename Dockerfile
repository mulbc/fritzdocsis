##
## Build
##
FROM golang:alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /fritzDocsis

##
## Deploy
##
FROM alpine

WORKDIR /

COPY --from=build /fritzDocsis /fritzDocsis

EXPOSE 2112

ENTRYPOINT ["./fritzDocsis"]