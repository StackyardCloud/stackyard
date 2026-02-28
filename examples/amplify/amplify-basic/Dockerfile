FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-basic ./examples/amplify/amplify-basic
ENTRYPOINT ["/amplify-basic"]
