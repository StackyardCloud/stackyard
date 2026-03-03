FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito-advanced ./examples/aws/cognito/cognito-advanced
ENTRYPOINT ["/cognito-advanced"]
