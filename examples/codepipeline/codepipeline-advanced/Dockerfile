FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codepipeline-advanced ./examples/codepipeline/codepipeline-advanced
ENTRYPOINT ["/codepipeline-advanced"]
