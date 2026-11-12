FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /communication-networktraversal-2022-03-01-preview ./examples/azure/communication-networktraversal-2022-03-01-preview

ENTRYPOINT ["/communication-networktraversal-2022-03-01-preview"]
