FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /trustedadvisor ./examples/aws/trustedadvisor
ENTRYPOINT ["/trustedadvisor"]
