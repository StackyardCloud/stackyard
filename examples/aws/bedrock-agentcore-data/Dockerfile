FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-data ./examples/aws/bedrock-agentcore-data
ENTRYPOINT ["/bedrock-agentcore-data"]
