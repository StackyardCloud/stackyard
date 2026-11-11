FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudcontrolapi ./examples/aws/cloudcontrolapi
ENTRYPOINT ["/cloudcontrolapi"]
