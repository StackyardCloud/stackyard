FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudformation-advanced ./examples/aws/cloudformation/cloudformation-advanced
ENTRYPOINT ["/cloudformation-advanced"]
