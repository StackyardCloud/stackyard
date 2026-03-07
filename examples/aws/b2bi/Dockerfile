FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /b2bi ./examples/aws/b2bi
ENTRYPOINT ["/b2bi"]
