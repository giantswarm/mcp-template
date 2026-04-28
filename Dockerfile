FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/giantswarm/mcp-template/cmd.version=${VERSION}" \
    -o /out/mcp-template .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/mcp-template /usr/local/bin/mcp-template
USER nonroot:nonroot
EXPOSE 8080 9091
ENTRYPOINT ["/usr/local/bin/mcp-template"]
CMD ["serve"]
