FROM golang:1.19-bullseye AS build

WORKDIR /app

COPY . /app

RUN go build -o /app/pulumiservice-exporter

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /app
COPY --from=build /app/pulumiservice-exporter /app/pulumiservice-exporter
EXPOSE 9414
USER nonroot:nonroot

ENTRYPOINT ["/pulumiservice-exporter"]
