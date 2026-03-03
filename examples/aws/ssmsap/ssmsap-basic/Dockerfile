FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ssmsap-basic ./examples/aws/ssmsap/ssmsap-basic
ENTRYPOINT ["/ssmsap-basic"]
