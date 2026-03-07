FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb ./examples/aws/odb
ENTRYPOINT ["/odb"]
