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
Der Importer versendet nach jeder Migration automatisch Benachrichtigungsmails. Das Ziel davon wird über die values.yaml gesteuert.
Standardmäßig ist es so konfiguriert, dass es auf einen Mailhog auf dem Host versendet wird. Dafür muss nichts konfiguriert werden.
Um den Mailhog zu starten, muss auf dem Host `docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog` ausgeführt werden.