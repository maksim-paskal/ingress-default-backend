# Install
```
kubectl -n $your_app_namespace apply -f deployment.yaml
```

Add to your ingress annotations
```
annotations:
  nginx.ingress.kubernetes.io/custom-http-errors: 500,502,503,504,507
  nginx.ingress.kubernetes.io/default-backend: ingress-default-backend
```

# Test
```
make test
curl -H "X-Code: 500" -H "X-Request-ID: 123" 127.0.0.1
```