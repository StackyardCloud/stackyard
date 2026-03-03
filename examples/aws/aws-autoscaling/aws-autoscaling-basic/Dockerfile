FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /aws-autoscaling-basic ./examples/aws/aws-autoscaling/aws-autoscaling-basic
ENTRYPOINT ["/aws-autoscaling-basic"]
