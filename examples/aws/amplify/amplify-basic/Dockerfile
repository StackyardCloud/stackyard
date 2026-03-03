FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-basic ./examples/aws/amplify/amplify-basic
ENTRYPOINT ["/amplify-basic"]
