FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2-autoscaling-basic ./examples/aws/ec2-autoscaling/ec2-autoscaling-basic
ENTRYPOINT ["/ec2-autoscaling-basic"]
