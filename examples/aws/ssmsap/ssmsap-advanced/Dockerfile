FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssmsap-advanced ./examples/aws/ssmsap/ssmsap-advanced
ENTRYPOINT ["/ssmsap-advanced"]
