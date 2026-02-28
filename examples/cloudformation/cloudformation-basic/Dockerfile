FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudformation-basic ./examples/cloudformation/cloudformation-basic
ENTRYPOINT ["/cloudformation-basic"]
