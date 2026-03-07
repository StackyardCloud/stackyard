FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /codepipeline ./examples/aws/codepipeline
ENTRYPOINT ["/codepipeline"]
