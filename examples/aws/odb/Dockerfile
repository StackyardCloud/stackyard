FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb ./examples/aws/odb
ENTRYPOINT ["/odb"]
