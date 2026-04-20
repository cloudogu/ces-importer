## Develop `ces-importer

## Install local helmet chart

### Create secrets
If not already created in another way, secrets must be created before installing the ces-importer.
must be created.
This can be done for local development by executing `make apikey-secret`.
If necessary, the values `IMPORTER_SSH_KEY_FILE` and `EXPORTER_API_KEY` in the .env file must first be adjusted to the desired values.
to the desired values.

### Installation
To install the ces-importer in the local k8s-ecosystem, the command `make helm-apply` can be executed.
can be executed.
To uninstall it again, the command `make helm-delete` can be used.

### Send mails
The importer can automatically send notification emails after each migration. The destination is controlled via values.yaml.
By default, email dispatch is deactivated. If enabled, it is configured to be sent to a mailhog on the host. Nothing more 
needs to be configured for this. To start the mailhog, `docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog` must be 
executed on the host.
Additionally, it can be configured whether the mail server uses implicit TLS, STARTTLS, or no TLS at all.

- implicit
- starttls
- none

### Trigger the migration manually
By creating a configmap with the label `k8s.cloudogu.com/start-migration` the migration will be triggered manually.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: start-migration
  labels:
    k8s.cloudogu.com/start-migration: "true"
```