FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-firewall ./examples/aws/network-firewall/network-firewall
ENTRYPOINT ["/network-firewall"]
