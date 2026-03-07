FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssmsap ./examples/aws/ssmsap
ENTRYPOINT ["/ssmsap"]
