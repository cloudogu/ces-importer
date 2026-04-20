# `ces-importer` entwickeln

## Lokales Helm-Chart installieren

### Secrets anlegen
Sofern nicht auf andere Art bereits angelegt, müssen vor der Installation des ces-importers noch secrets angelegt 
werden.
Das kann für die Lokale Entwicklung durch das Ausführen von `make apikey-secret` erfolgen.
Gegebenenfalls müssen vorher noch die Werte `IMPORTER_SSH_KEY_FILE` und `EXPORTER_API_KEY` in der .env-Datei auf die 
gewünschten Werte angepasst werden.

### Installation
Um den ces-importer im lokalen k8s-ecosystem zu installieren, kann der Befehl `make helm-apply` ausgeführt 
werden.
Um ihn wieder zu deinstallieren, kann der Befehl `make helm-delete` verwendet werden.

### Mails versenden
Der Importer kann nach jeder Migration automatisch Benachrichtigungsmails versenden. Die Konfiguration der Empfänger 
erfolgt über die values.yaml Datei. Standardmäßig ist der Mailversand deaktiviert. Wird er aktiviert, so ist er so 
konfiguriert, dass es auf einen Mailhog auf dem Host versendet wird. Dafür muss nichts weiter konfiguriert werden. 
Um den Mailhog zu starten, muss auf dem Host`docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog` ausgeführt werden.
Es kann darüber hinaus konfiguriert werden, ob der Mailserver impilizites TLS, STARTTLS oder garkein TLS verwendet.
- implicit
- starttls
- none

### Migration manuell starten
Durch das Anlegen einer Configmap mit dem Label `k8s.cloudogu.com/start-migration` kann eine Migration manuell 
gestartet werden.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: start-migration
  labels:
    k8s.cloudogu.com/start-migration: "true"
```