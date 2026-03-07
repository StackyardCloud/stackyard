FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolapi ./examples/aws/cloudcontrolapi
ENTRYPOINT ["/cloudcontrolapi"]
