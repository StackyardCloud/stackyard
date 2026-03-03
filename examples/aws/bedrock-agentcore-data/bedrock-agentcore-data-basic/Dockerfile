FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /bedrock-agentcore-data-basic ./examples/aws/bedrock-agentcore-data/bedrock-agentcore-data-basic
ENTRYPOINT ["/bedrock-agentcore-data-basic"]
