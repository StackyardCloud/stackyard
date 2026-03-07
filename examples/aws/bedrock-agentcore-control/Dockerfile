FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-control ./examples/aws/bedrock-agentcore-control
ENTRYPOINT ["/bedrock-agentcore-control"]
