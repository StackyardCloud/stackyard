FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudformation ./examples/aws/cloudformation/cloudformation
ENTRYPOINT ["/cloudformation"]
