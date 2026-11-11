FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2 ./examples/aws/ec2
ENTRYPOINT ["/ec2"]
