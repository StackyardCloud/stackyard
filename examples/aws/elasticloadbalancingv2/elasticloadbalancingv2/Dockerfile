FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancingv2 ./examples/aws/elasticloadbalancingv2/elasticloadbalancingv2
ENTRYPOINT ["/elasticloadbalancingv2"]
