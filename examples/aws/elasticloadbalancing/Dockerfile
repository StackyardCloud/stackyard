FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancing ./examples/aws/elasticloadbalancing
ENTRYPOINT ["/elasticloadbalancing"]
