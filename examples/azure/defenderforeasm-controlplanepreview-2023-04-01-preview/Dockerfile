FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /defenderforeasm-controlplanepreview-2023-04-01-preview-example ./examples/azure/defenderforeasm-controlplanepreview-2023-04-01-preview

ENTRYPOINT ["/defenderforeasm-controlplanepreview-2023-04-01-preview-example"]
