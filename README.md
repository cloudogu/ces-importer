# CES-Importer

### Build & Push Container

```shell
docker build -t registry.cloudogu.com/testing/ces-importer:0.0.1 . 

docker push registry.cloudogu.com/testing/ces-importer:0.0.1
```

### Build & Push Helm-Chart

```shell
make helm-package

helm push target/k8s/helm/ces-importer-0.0.1.tgz oci://registry.cloudogu.com/testing/ces-importer-helm
```

### Install in k8s

```shell
# template only
helm template -n ecosystem -f myvalues.yaml --set-file secret.privateKey=/path/to/private.key ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

# install
helm install -n ecosystem -f myvalues.yaml --set-file secret.privateKey=/path/to/private.key ces-importer oci://registry.cloudogu.com/testing/ces-importer-helm/ces-importer --version 0.0.1

helm uninstall -n ecosystem ces-importer
```

##### Exmaple `myvalues.yaml`
```yaml
config:
  log_level: "DEBUG"
  exporter_host: "classic-ces.exporter"
  exporter_port: "7000"
  exporter_ssh_user: "ces-exporter"
```