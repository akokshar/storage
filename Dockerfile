FROM golang:latest as builder
WORKDIR /go/src/github.com/akokshar/storage
COPY . .
RUN go build -a -race -o app .

FROM fedora:latest
WORKDIR /root
COPY --from=builder /go/src/github.com/akokshar/storage/app .
CMD ["./app"]
