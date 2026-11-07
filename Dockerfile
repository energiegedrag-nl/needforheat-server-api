FROM --platform=$BUILDPLATFORM golang:1.27 AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/needforheat-server-api

# Create /data folder to be copied later.
RUN mkdir /data

# Download dependencies.
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /go/bin/needforheat-server-api .

FROM gcr.io/distroless/static-debian13:nonroot

# Copy /data folder with correct permissions.
COPY --from=build --chown=nonroot /data /data

# Copy binary.
COPY --from=build /go/bin/needforheat-server-api /usr/bin/

USER nonroot

VOLUME /data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --start-interval=2s --retries=3 \
    CMD ["needforheat-server-api", "healthcheck"]

ENTRYPOINT ["needforheat-server-api"]
CMD ["serve"]
