FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssmsap ./examples/aws/ssmsap/ssmsap
ENTRYPOINT ["/ssmsap"]
