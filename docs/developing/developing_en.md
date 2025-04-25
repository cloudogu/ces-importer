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
