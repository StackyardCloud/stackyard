FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /waf ./examples/aws/waf/waf
ENTRYPOINT ["/waf"]

