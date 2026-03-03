FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /acm-basic ./examples/aws/acm/acm-basic
ENTRYPOINT ["/acm-basic"]
