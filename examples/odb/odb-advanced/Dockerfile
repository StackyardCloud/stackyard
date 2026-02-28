FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb-advanced ./examples/odb/odb-advanced
ENTRYPOINT ["/odb-advanced"]
