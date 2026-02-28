FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2-autoscaling-advanced ./examples/ec2-autoscaling/ec2-autoscaling-advanced
ENTRYPOINT ["/ec2-autoscaling-advanced"]
