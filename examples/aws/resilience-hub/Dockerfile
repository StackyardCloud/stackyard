FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /resilience-hub ./examples/aws/resilience-hub
ENTRYPOINT ["/resilience-hub"]
