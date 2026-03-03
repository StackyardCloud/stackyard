FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolapi ./examples/aws/cloudcontrolapi/cloudcontrolapi
ENTRYPOINT ["/cloudcontrolapi"]
