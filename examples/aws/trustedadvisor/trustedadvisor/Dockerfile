FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /trustedadvisor ./examples/aws/trustedadvisor/trustedadvisor
ENTRYPOINT ["/trustedadvisor"]
