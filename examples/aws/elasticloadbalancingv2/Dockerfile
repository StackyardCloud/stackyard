FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancingv2 ./examples/aws/elasticloadbalancingv2
ENTRYPOINT ["/elasticloadbalancingv2"]
