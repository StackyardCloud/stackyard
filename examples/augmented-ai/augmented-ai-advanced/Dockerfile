FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /augmented-ai-advanced ./examples/augmented-ai/augmented-ai-advanced
ENTRYPOINT ["/augmented-ai-advanced"]
