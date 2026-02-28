FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /amplify-advanced ./examples/amplify/amplify-advanced
ENTRYPOINT ["/amplify-advanced"]
