FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticloadbalancingv2-basic ./examples/elasticloadbalancingv2/elasticloadbalancingv2-basic
ENTRYPOINT ["/elasticloadbalancingv2-basic"]
