FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /network-firewall-basic ./examples/network-firewall/network-firewall-basic
ENTRYPOINT ["/network-firewall-basic"]
