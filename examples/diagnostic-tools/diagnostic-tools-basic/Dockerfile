FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /diagnostic-tools-basic ./examples/diagnostic-tools/diagnostic-tools-basic
ENTRYPOINT ["/diagnostic-tools-basic"]
