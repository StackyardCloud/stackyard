FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancingv2-advanced ./examples/aws/elasticloadbalancingv2/elasticloadbalancingv2-advanced
ENTRYPOINT ["/elasticloadbalancingv2-advanced"]
