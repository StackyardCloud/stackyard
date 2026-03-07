FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /pcs ./examples/aws/pcs
ENTRYPOINT ["/pcs"]
