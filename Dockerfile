FROM golang:latest as builder
WORKDIR /go/src/github.com/akokshar/storage
COPY . .
RUN CGO_ENABLED=0 go build -o app .

FROM scratch

COPY --from=builder /go/src/github.com/akokshar/storage/app /

ENV BASEDIR /Store
ENV PORT 8080

CMD ["/app"]
