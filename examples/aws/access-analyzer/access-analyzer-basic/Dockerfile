FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /access-analyzer-basic ./examples/aws/access-analyzer/access-analyzer-basic
CMD ["/access-analyzer-basic"]
