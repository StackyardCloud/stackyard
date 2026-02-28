FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /diagnostic-tools-advanced ./examples/diagnostic-tools/diagnostic-tools-advanced
ENTRYPOINT ["/diagnostic-tools-advanced"]
