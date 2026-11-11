FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /defenderforeasm-dataplanepreview-2024-10-01-preview-example ./examples/azure/defenderforeasm-dataplanepreview-2024-10-01-preview

ENTRYPOINT ["/defenderforeasm-dataplanepreview-2024-10-01-preview-example"]
