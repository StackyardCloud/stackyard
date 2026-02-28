FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-advanced ./examples/singlesignon/singlesignon-advanced
ENTRYPOINT ["/singlesignon-advanced"]
