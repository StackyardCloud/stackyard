FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /waf-advanced ./examples/waf/waf-advanced
ENTRYPOINT ["/waf-advanced"]

