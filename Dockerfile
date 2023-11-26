FROM --platform=$BUILDPLATFORM golang:1.21.4 AS BUILD
WORKDIR /app
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=development
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s -X 'main.Version=${VERSION}'" .

FROM scratch
WORKDIR /app
COPY --from=BUILD /app/hubble-drop-eventer /app

CMD [ "/app/hubble-drop-eventer" ]
