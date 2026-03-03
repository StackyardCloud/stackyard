FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb ./examples/aws/odb/odb
ENTRYPOINT ["/odb"]
