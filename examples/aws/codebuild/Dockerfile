FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /codebuild ./examples/aws/codebuild
ENTRYPOINT ["/codebuild"]
