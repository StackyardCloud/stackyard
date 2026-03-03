FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codepipeline ./examples/aws/codepipeline/codepipeline
ENTRYPOINT ["/codepipeline"]
