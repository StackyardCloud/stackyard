FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb-advanced ./examples/aws/odb/odb-advanced
ENTRYPOINT ["/odb-advanced"]
