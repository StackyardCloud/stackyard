FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-data-advanced ./examples/bedrock-agentcore-data/bedrock-agentcore-data-advanced
ENTRYPOINT ["/bedrock-agentcore-data-advanced"]
