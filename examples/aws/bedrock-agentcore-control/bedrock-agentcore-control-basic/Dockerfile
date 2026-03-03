FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-control-basic ./examples/aws/bedrock-agentcore-control/bedrock-agentcore-control-basic
ENTRYPOINT ["/bedrock-agentcore-control-basic"]
