FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito ./examples/aws/cognito
ENTRYPOINT ["/cognito"]
