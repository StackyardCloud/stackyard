FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancing-basic ./examples/aws/elasticloadbalancing/elasticloadbalancing-basic
ENTRYPOINT ["/elasticloadbalancing-basic"]
