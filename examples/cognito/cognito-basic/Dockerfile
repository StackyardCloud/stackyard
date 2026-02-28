FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cognito-basic ./examples/cognito/cognito-basic
ENTRYPOINT ["/cognito-basic"]
