FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resilience-hub ./examples/aws/resilience-hub/resilience-hub
ENTRYPOINT ["/resilience-hub"]
