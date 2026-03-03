FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito ./examples/aws/cognito/cognito
ENTRYPOINT ["/cognito"]
