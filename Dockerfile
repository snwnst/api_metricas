FROM golang:1.12.6 as builder

WORKDIR /app
COPY ./src .

RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/exe_slave_api .

FROM alpine:3.10.0  

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /go/bin/exe_slave_api .

EXPOSE 80

CMD ["./exe_slave_api"] 