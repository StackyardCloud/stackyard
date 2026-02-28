FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /resilience-hub-basic ./examples/resilience-hub/resilience-hub-basic
ENTRYPOINT ["/resilience-hub-basic"]
