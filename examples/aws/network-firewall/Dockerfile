FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-firewall ./examples/aws/network-firewall
ENTRYPOINT ["/network-firewall"]
