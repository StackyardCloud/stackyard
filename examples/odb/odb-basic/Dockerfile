FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /odb-basic ./examples/odb/odb-basic
ENTRYPOINT ["/odb-basic"]
