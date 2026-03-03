FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify ./examples/aws/amplify/amplify
ENTRYPOINT ["/amplify"]
