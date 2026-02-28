FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /singlesignon-basic ./examples/singlesignon/singlesignon-basic
ENTRYPOINT ["/singlesignon-basic"]
