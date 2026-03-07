FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /waf ./examples/aws/waf
ENTRYPOINT ["/waf"]

