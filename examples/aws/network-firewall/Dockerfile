FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-firewall ./examples/aws/network-firewall
ENTRYPOINT ["/network-firewall"]
