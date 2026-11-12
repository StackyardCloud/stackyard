FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /dlm ./examples/aws/dlm
CMD ["/dlm"]
