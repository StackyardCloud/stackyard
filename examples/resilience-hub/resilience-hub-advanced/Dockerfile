FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resilience-hub-advanced ./examples/resilience-hub/resilience-hub-advanced
ENTRYPOINT ["/resilience-hub-advanced"]
