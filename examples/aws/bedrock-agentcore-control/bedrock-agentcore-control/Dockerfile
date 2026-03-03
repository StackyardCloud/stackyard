FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-control ./examples/aws/bedrock-agentcore-control/bedrock-agentcore-control
ENTRYPOINT ["/bedrock-agentcore-control"]
