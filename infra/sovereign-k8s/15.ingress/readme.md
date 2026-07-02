# Ingress and SSL Certificate Setup

## Install NGINX Ingress Controller
To install the NGINX ingress controller on your cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.0/deploy/static/provider/cloud/deploy.yaml
```

## Install Cert-Manager
To deploy cert-manager for automatic TLS certificates:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.4/cert-manager.yaml
```

Once installed, apply the `issuer.yaml` configuration to register the Let's Encrypt production issuer.
