FROM alpine

WORKDIR /

# Get binary from goreleaser
COPY fritzdocsis /

EXPOSE 2112

ENTRYPOINT ["./fritzDocsis"]