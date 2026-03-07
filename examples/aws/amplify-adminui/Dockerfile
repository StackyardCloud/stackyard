FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-adminui ./examples/aws/amplify-adminui
ENTRYPOINT ["/amplify-adminui"]
