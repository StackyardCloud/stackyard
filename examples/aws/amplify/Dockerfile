FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify ./examples/aws/amplify
ENTRYPOINT ["/amplify"]
