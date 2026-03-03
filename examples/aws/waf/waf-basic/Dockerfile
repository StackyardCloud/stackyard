FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /waf-basic ./examples/aws/waf/waf-basic
ENTRYPOINT ["/waf-basic"]

