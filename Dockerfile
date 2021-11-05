FROM alpine:latest

COPY ./ingress-default-backend /app/ingress-default-backend
COPY ./templates/ /app/templates/

WORKDIR /app

ENTRYPOINT [ "/app/ingress-default-backend" ]