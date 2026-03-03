FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /augmented-ai-basic ./examples/aws/augmented-ai/augmented-ai-basic
ENTRYPOINT ["/augmented-ai-basic"]
