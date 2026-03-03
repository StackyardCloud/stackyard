FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /b2bi-basic ./examples/aws/b2bi/b2bi-basic
ENTRYPOINT ["/b2bi-basic"]
