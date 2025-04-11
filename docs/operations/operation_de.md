# Ausführen von `ces-importer`

## Vor dem Start

Vor dem Einsatz sind einige Voraussetzungen zu erfüllen:

- ein k8s Secret, das den API-Schlüssel des Exportersystems enthält
   - der Name des Geheimnisses und das Datenfeld, das den Wert enthalten wird, müssen in diesen beiden Werten konfiguriert werden:
      - `config.exporter.apiSecretName` enthält den Namen der k8s Secret, z. B. `ces-exporter-secret`
      - `config.exporter.apiSecretDataKey` enthält den Namen des Schlüssels, der den Wert `apiKey` enthält
   - Beispiel für die Erstellung:
     `kubectl -n ecosystem create secret generic ces-exporter-secret --from-literal=apiKey=ApiKey-example-123`
- ein k8s Secret, das den privaten SSH-Schlüssel enthält, der für den Zugriff auf das Exporter-System über SSH verwendet wird
   - der Name des Geheimnisses und das Datenfeld, das den Wert enthält, müssen in diesen beiden Werten konfiguriert werden:
      - `config.importer.sshSecretName` enthält den Namen der k8s Secret, z. B. `ces-importer-secret`
      - `config.importer.sshSecretDataKey` enthält den Namen des Schlüssels, der den Wert `privateKey` enthält
   - Beispiel für die Erstellung:
     `kubectl -n ecosystem create secret generic ces-importer-secret --from-file=privateKey=yourPrivateKeyHere`
- eine Helm Chart `values.yaml` mit den relevanten Konfigurationspunkten befüllen 

## Bereitstellung der Anwendung mit Helm


```shell
# template only
helm template -n ecosystem -f myvalues.yaml ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

# install
helm install -n ecosystem -f myvalues.yaml ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

helm uninstall -n ecosystem ces-importer
```

## Example `values.yaml`

Das folgende Snippet ist ein abgekürzter Ausschnitt einer möglichen `values.yaml`. Für eine vollständige Version konsultieren Sie bitte die vollständige YAML-Datei (diese enthält auch Kommentare) im Dateipfad `k8s/helm/values.yaml`.

```yaml
main:
  image: registry.cloudogu.com/testing/ces-importer:0.0.1
  imagePullSecrets:
    - name: "ces-exporter-registries"
  imagePullPolicy: IfNotPresent
job:
  image: registry.cloudogu.com/testing/ces-importer:0.0.1
  imagePullSecrets:
    - name: "ces-exporter-registries"
config:
  exporter:
    host: my.classic.ces.exporter.net
    apiSecretName: "ces-exporter-secret"
    apiSecretDataKey: "apiKey"
  importer:
    sshSecretName: "ces-importer-secret"
    sshSecretDataKey: "privateKey"
  migration:
    regularSchedule: "0 4 * * *"
    finalSchedule: "2025-04-03 12:34:56Z"
```

Im Kontext der Konfiguration ergibt sich in etwa ein solches Zielbild von `ces-importer`:

![Ein Diagram zweier Systeme "exporter" als Quelle und "importer" als Ziel. Personen mit der Administratorrolle wenden ein Helm Chart und Secrets auf den Importer an, worauf die ces-importer seine Arbeit aufnehmen kann. Als erstes wird ein Endpunkt bzgl systemspezifischer Exporter-Host-Konfigurationen abgefragt. Dann wird eine Job-Resource erzeugt, die die eigentliche Importierarbeit übernimmt.](images/ces-importer.drawio.png "Konfigurationspunkte von ces-importer und wie diese zur Laufzeit angewendet werden.")
