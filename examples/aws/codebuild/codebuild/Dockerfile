FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codebuild ./examples/aws/codebuild/codebuild
ENTRYPOINT ["/codebuild"]
