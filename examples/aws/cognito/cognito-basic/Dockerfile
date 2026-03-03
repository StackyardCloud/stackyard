FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito-basic ./examples/aws/cognito/cognito-basic
ENTRYPOINT ["/cognito-basic"]
