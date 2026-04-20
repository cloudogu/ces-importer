### Mailpit aufsetzen

Unter Treafik kann ein mailpit für Testzwecke aufgesetzt und konfiguriert werden.
Dazu muss zunächst ein entsprechendes Zertifikat erstellt werden um mailpit mit STARTTLS verwenden zu können:

#### Zertifikat:
Server-Konfiguration mit MailPit Alternative-Names erstellen: server.cnf

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

Zertifikat mit Konfiguration erstellen

```bash
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -config server.cnf
openssl x509 -req   -in server.csr   -CA ca.crt   -CAkey ca.key   -CAcreateserial   -out server.crt   -days 825   -sha256   -extensions req_ext   -extfile server.cnf
```

Prüfen ob Alternative-Names enthalten sind:
```bash
openssl x509 -in server.crt -text -noout
```

#### Zertifikat als Config-Map für Mailpit im Cluster ablegen
```bash
kubectl create secret tls mailpit-tls --cert=server.crt --key=server.key
```

#### Zertifikat als Config-Map für den Importer im Cluster ablegen
```bash
kubectl create configmap ces-importer-mail-ca --from-file=mail.crt=server.crt --namespace=ecosystem
```

#### Mailpit als Deployment ablegen

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

#### Testen über Portfowarding

- via k9s den Port 8025 am Mailpit-Pod freigeben
- ggf. den Port 8025 im InteliJ von coder freigeben