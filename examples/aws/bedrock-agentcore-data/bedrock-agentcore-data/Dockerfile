FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-data ./examples/aws/bedrock-agentcore-data/bedrock-agentcore-data
ENTRYPOINT ["/bedrock-agentcore-data"]
