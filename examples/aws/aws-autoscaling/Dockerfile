FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-autoscaling ./examples/aws/aws-autoscaling
ENTRYPOINT ["/aws-autoscaling"]
