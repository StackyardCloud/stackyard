FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-firewall-advanced ./examples/network-firewall/network-firewall-advanced
ENTRYPOINT ["/network-firewall-advanced"]
