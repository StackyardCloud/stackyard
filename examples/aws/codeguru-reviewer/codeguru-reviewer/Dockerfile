FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-reviewer ./examples/aws/codeguru-reviewer/codeguru-reviewer
ENTRYPOINT ["/codeguru-reviewer"]
