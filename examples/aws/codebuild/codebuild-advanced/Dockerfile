FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codebuild-advanced ./examples/aws/codebuild/codebuild-advanced
ENTRYPOINT ["/codebuild-advanced"]
