FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /trustedadvisor-advanced ./examples/trustedadvisor/trustedadvisor-advanced
ENTRYPOINT ["/trustedadvisor-advanced"]
