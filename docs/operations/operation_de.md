# Ausführen von `ces-importer`

## Vor dem Start

Vor dem Einsatz sind einige Voraussetzungen zu erfüllen:

- ein k8s Secret, das den API-Schlüssel des Exportersystems enthält
   - der Name des Geheimnisses und das Datenfeld, das den Wert enthalten wird, müssen in diesen beiden Werten konfiguriert werden:
      - `config.exporter.apiSecretName` enthält den Namen der k8s Secret, z. B. `ces-exporter-secret`.
      - `config.exporter.apiSecretDataKey` enthält den Namen des Schlüssels, der den Wert `apiKey` enthält
   - Beispiel für die Erstellung:
     `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
- ein k8s Secret, das den privaten SSH-Schlüssel enthält, der für den Zugriff auf das Exporter-System über SSH verwendet wird
   - der Name des Geheimnisses und das Datenfeld, das den Wert enthält, müssen in diesen beiden Werten konfiguriert werden:
      - `config.importer.sshSecretName` enthält den Namen der k8s Secret, z. B. `ces-importer-secret`.
      - `config.importer.sshSecretDataKey` enthält den Namen des Schlüssels, der den Wert `privateKey` enthält
   - Beispiel für die Erstellung:
     `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`


