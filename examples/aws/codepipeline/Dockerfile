FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /codepipeline ./examples/aws/codepipeline
ENTRYPOINT ["/codepipeline"]
