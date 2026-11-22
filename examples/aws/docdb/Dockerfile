FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /docdb ./examples/aws/docdb
ENTRYPOINT ["/docdb"]
