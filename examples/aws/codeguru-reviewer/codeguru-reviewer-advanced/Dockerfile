FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codeguru-reviewer-advanced ./examples/aws/codeguru-reviewer/codeguru-reviewer-advanced
ENTRYPOINT ["/codeguru-reviewer-advanced"]
