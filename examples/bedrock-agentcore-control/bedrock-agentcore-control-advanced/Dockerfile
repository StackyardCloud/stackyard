FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-control-advanced ./examples/bedrock-agentcore-control/bedrock-agentcore-control-advanced
ENTRYPOINT ["/bedrock-agentcore-control-advanced"]
