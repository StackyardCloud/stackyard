FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /diagnostic-tools ./examples/aws/diagnostic-tools/diagnostic-tools
ENTRYPOINT ["/diagnostic-tools"]
