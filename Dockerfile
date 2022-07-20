FROM alpine:latest

COPY ./ingress-default-backend /app/ingress-default-backend
COPY ./templates/ /app/templates/

WORKDIR /app

RUN apk upgrade \
&& addgroup -g 30001 -S app \
&& adduser -u 30001 -D -S -G app app

USER 30001

ENTRYPOINT [ "/app/ingress-default-backend" ]