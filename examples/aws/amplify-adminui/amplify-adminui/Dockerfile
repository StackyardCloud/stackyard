FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-adminui ./examples/aws/amplify-adminui/amplify-adminui
ENTRYPOINT ["/amplify-adminui"]
