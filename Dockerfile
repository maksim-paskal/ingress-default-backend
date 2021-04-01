FROM golang:1.16 as build

COPY *.go /usr/src/ingress-default-backend/
COPY go.* /usr/src/ingress-default-backend/
COPY ./cmd /usr/src/ingress-default-backend/cmd

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GOFLAGS="-trimpath"

RUN cd /usr/src/ingress-default-backend \
  && go mod download \
  && go mod verify \
  && go build -v -o ingress-default-backend -ldflags "-X main.buildTime=$(date +"%Y%m%d%H%M%S")" ./cmd

FROM alpine:3

COPY --from=build /usr/src/ingress-default-backend/ingress-default-backend /app/ingress-default-backend
COPY ./templates/ /app/templates/

WORKDIR /app
CMD /app/ingress-default-backend