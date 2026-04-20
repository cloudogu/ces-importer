### Set up Mailpit

Under Traefik, a Mailpit instance can be set up and configured for testing purposes.
First, a corresponding certificate must be created in order to use Mailpit with STARTTLS:

#### Certificate:

Create a server configuration with MailPit alternative names: server.cnf
```
[ req ]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[ dn ]
CN = mailpit

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = mailpit
DNS.2 = mailpit.ecosystem.svc.cluster.local
DNS.3 = localhost
```

Crea```te the certificate using the configuration:
```bash
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -config server.cnf
openssl x509 -req   -in server.csr   -CA ca.crt   -CAkey ca.key   -CAcreateserial   -out server.crt   -days 825   -sha256   -extensions req_ext   -extfile server.cnf
```

Verify that alternative names are included:

```bash
openssl x509 -in server.crt -text -noout
```

### Store certificate as a secret for Mailpit in the cluster
```bash
kubectl create secret tls mailpit-tls --cert=server.crt --key=server.key
```

### Store certificate as a ConfigMap for the importer in the cluster

```
kubectl create configmap ces-importer-mail-ca --from-file=mail.crt=server.crt --namespace=ecosystem
```


### Deploy Mailpit

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mailpit
  labels:
    app: mailpit
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mailpit
  template:
    metadata:
      labels:
        app: mailpit
    spec:
      containers:
        - name: mailpit
          image: axllent/mailpit:latest
          ports:
            - containerPort: 8025  # Web UI
            - containerPort: 1025  # SMTP
          env:
            - name: MP_MAX_MESSAGES
              value: "1000"
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi
          args:
            - "--smtp-tls-cert=/certs/tls.crt"
            - "--smtp-tls-key=/certs/tls.key"
          volumeMounts:
            - name: tls
              mountPath: /certs
              readOnly: true
      volumes:
        - name: tls
          secret:
            secretName: mailpit-tls
---
apiVersion: v1
kind: Service
metadata:
  name: mailpit
  labels:
    app: mailpit
spec:
  selector:
    app: mailpit
  ports:
    - name: http
      port: 8025
      targetPort: 8025
    - name: smtp
      port: 1025
      targetPort: 1025
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mailpit
  annotations:
    kubernetes.io/ingress.class: traefik
spec:
  rules:
    - host: mailpit.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: mailpit
                port:
                  number: 8025
```

### Testing via port forwarding
- Use k9s to expose port 8025 on the Mailpit pod
- If necessary, expose port 8025 in IntelliJ (coder environment)