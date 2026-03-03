FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-autoscaling-advanced ./examples/aws/aws-autoscaling/aws-autoscaling-advanced
ENTRYPOINT ["/aws-autoscaling-advanced"]
