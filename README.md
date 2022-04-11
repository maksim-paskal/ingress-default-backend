# Install

```bash
helm repo add maksim-paskal-ingress-default-backend https://maksim-paskal.github.io/ingress-default-backend/

helm repo update

helm upgrade ingress-default-backend \
--install \
--create-namespace \
--namespace ingress-default-backend \
maksim-paskal-ingress-default-backend/ingress-default-backend
```

Add to your ingress annotations

```yaml
annotations:
  nginx.ingress.kubernetes.io/custom-http-errors: 500,502,503,504,507
  nginx.ingress.kubernetes.io/default-backend: ingress-default-backend
```

## Test

```bash
make run
curl -H "X-Code: 500" -H "X-Request-ID: 123" 127.0.0.1:8080
```
