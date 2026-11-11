FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /diagnostic-tools ./examples/aws/diagnostic-tools
ENTRYPOINT ["/diagnostic-tools"]
