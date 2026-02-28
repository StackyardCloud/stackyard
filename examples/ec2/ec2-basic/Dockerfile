FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ec2-basic ./examples/ec2/ec2-basic
ENTRYPOINT ["/ec2-basic"]
