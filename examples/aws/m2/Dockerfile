FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /m2 ./examples/aws/m2
ENTRYPOINT ["/m2"]
