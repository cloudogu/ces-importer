# Ausführen von `ces-importer`

## Vor dem Start

Vor dem Einsatz sind einige Voraussetzungen zu erfüllen:

- K8s Secrets:
  - eins, das den API-Schlüssel des Exportersystems enthält
     - der Name des Geheimnisses und das Datenfeld, das den Wert enthalten wird, müssen in diesen beiden Werten konfiguriert werden:
        - `config.exporter.api.secretName` enthält den Namen der k8s Secret, z. B. `ces-exporter-secret`
        - `config.exporter.api.secretDataKey` enthält den Namen des Schlüssels, der den Wert `apiKey` enthält
     - Beispiel für die Erstellung:
       `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
  - eins, das den privaten SSH-Schlüssel enthält, der für den Zugriff auf das Exporter-System über SSH verwendet wird
     - der Name des Geheimnisses und das Datenfeld, das den Wert enthält, müssen in diesen beiden Werten konfiguriert werden:
        - `config.importer.ssh.secretName` enthält den Namen der k8s Secret, z. B. `ces-importer-secret`
        - `config.importer.ssh.secretDataKey` enthält den Namen des Schlüssels, der den Wert `privateKey` enthält
     - Beispiel für die Erstellung:
       `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`
  - eins, das das Passwort für den Mail-Server enthält
      - der Name des Geheimnisses und das Datenfeld, das den Wert enthält, müssen in diesen beiden Werten konfiguriert werden:
          - `config.importer.smtp.secretName` enthält den Namen der k8s Secret, z. B. `ces-importer-secret`
          - `config.importer.smtp.secretDataKey` enthält den Namen des Schlüssels, der den Wert `mailPassword` enthält
      - Beispiel für die Erstellung:
        `kubectl -n ecosystem create secret generic ces-importer-secret --from-literal=mailPassword=yourMailPasswordHere`
  - eins, das das Image-Pull-Secret für das ces-importer-Deployment enthält
    - dieses muss dann auch in der `values.yaml` angegeben werden
- eine Helm Chart `values.yaml` mit den relevanten Konfigurationspunkten befüllen 

## Installation as a CES component

The ces-importer can be installed and configured as a CES component.
An example can be seen here:

```yaml
apiVersion: k8s.cloudogu.com/v1
kind: Component
metadata:
  name: ces-importer
  namespace: ecosystem
spec:
  name: ces-importer
  namespace: k8s
  version: 0.0.1
  valuesYamlOverwrite: |
    config:
      logging:
        level: DEBUG
      api:
        host: classic-ces.exporter
        skipTLSVerify: false
      migration:
        regularSchedule: "0/5 * * * *"
        finalSchedule: ""
        changeFQDN: false
```

Im Kontext der Konfiguration ergibt sich in etwa ein solches Zielbild von `ces-importer`:

![Ein Diagram zweier Systeme "exporter" als Quelle und "importer" als Ziel. Personen mit der Administratorrolle wenden ein Helm Chart und Secrets auf den Importer an, worauf die ces-importer seine Arbeit aufnehmen kann. Als erstes wird ein Endpunkt bzgl systemspezifischer Exporter-Host-Konfigurationen abgefragt. Dann wird eine Job-Resource erzeugt, die die eigentliche Importierarbeit übernimmt.](images/ces-importer.drawio.png "Konfigurationspunkte von ces-importer und wie diese zur Laufzeit angewendet werden.")
