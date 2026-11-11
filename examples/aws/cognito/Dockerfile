FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito ./examples/aws/cognito
ENTRYPOINT ["/cognito"]
