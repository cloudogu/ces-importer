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

##### Example `myvalues.yaml`
```yaml
config:
  log_level: "DEBUG"
  exporter_host: "classic-ces.exporter"
  exporter_port: "7000"
  exporter_ssh_user: "ces-exporter"
```

---
### What is the Cloudogu EcoSystem?
The Cloudogu EcoSystem is an open platform, which lets you choose how and where your team creates great software. Each service or tool is delivered as a Dogu, a Docker container. Each Dogu can easily be integrated in your environment just by pulling it from our registry. We have a growing number of ready-to-use Dogus, e.g. SCM-Manager, Jenkins, Nexus, SonarQube, Redmine and many more. Every Dogu can be tailored to your specific needs. Take advantage of a central authentication service, a dynamic navigation, that lets you easily switch between the web UIs and a smart configuration magic, which automatically detects and responds to dependencies between Dogus. The Cloudogu EcoSystem is open source and it runs either on-premises or in the cloud. The Cloudogu EcoSystem is developed by Cloudogu GmbH under [MIT License](https://cloudogu.com/license.html).

### How to get in touch?
Want to talk to the Cloudogu team? Need help or support? There are several ways to get in touch with us:

* [Website](https://cloudogu.com)
* [myCloudogu-Forum](https://forum.cloudogu.com/topic/34?ctx=1)
* [Email hello@cloudogu.com](mailto:hello@cloudogu.com)

---
&copy; 2025 Cloudogu GmbH - MADE WITH :heart:&nbsp;FOR DEV ADDICTS. [Legal notice / Impressum](https://cloudogu.com/imprint.html)
