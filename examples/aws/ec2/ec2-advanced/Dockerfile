FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2-advanced ./examples/aws/ec2/ec2-advanced
ENTRYPOINT ["/ec2-advanced"]
