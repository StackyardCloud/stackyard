FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudformation ./examples/aws/cloudformation
ENTRYPOINT ["/cloudformation"]
