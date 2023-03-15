FROM golang:1.19-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY main.go ./

RUN go build -o /pulumiservice-exporter

## Deploy
FROM gcr.io/distroless/base-debian10

WORKDIR /
COPY --from=build /pulumiservice-exporter /pulumiservice-exporter
EXPOSE 9414
USER nonroot:nonroot

ENTRYPOINT ["/pulumiservice-exporter"]
