FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2-autoscaling ./examples/aws/ec2-autoscaling
ENTRYPOINT ["/ec2-autoscaling"]
