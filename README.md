```
kubectl apply -f deployment.yaml
```

annotations:
  nginx.ingress.kubernetes.io/custom-http-errors: 500,502,503,504,507
  nginx.ingress.kubernetes.io/default-backend: ingress-default-backend