FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /augmented-ai ./examples/aws/augmented-ai
ENTRYPOINT ["/augmented-ai"]
