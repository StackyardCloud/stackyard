FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /augmented-ai ./examples/aws/augmented-ai/augmented-ai
ENTRYPOINT ["/augmented-ai"]
