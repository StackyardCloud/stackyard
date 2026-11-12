FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudhsm ./examples/aws/cloudhsm
ENTRYPOINT ["/cloudhsm"]
