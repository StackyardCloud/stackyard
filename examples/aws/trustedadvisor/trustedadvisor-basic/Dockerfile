FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /trustedadvisor-basic ./examples/aws/trustedadvisor/trustedadvisor-basic
ENTRYPOINT ["/trustedadvisor-basic"]
