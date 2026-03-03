FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /codedeploy-advanced ./examples/aws/codedeploy/codedeploy-advanced
ENTRYPOINT ["/codedeploy-advanced"]
