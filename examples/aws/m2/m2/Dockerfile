FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /m2 ./examples/aws/m2/m2
ENTRYPOINT ["/m2"]
