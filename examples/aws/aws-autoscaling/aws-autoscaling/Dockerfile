FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-autoscaling ./examples/aws/aws-autoscaling/aws-autoscaling
ENTRYPOINT ["/aws-autoscaling"]
