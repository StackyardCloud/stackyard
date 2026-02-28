FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codebuild-basic ./examples/codebuild/codebuild-basic
ENTRYPOINT ["/codebuild-basic"]
