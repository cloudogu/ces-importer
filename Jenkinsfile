#!groovy
@Library(['github.com/cloudogu/ces-build-lib@4.2.0', 'github.com/cloudogu/dogu-build-lib@v3.2.0'])
import com.cloudogu.ces.cesbuildlib.*
import com.cloudogu.ces.dogubuildlib.*

Git git = new Git(this, "cesmarvin")
git.committerName = 'cesmarvin'
git.committerEmail = 'cesmarvin@cloudogu.com'
gitflow = new GitFlow(this, git)
github = new GitHub(this, git)
changelog = new Changelog(this)
goVersion = "1.25.5"

// Configuration of repository
String repositoryName = "ces-importer"

// Configuration of branches
productionReleaseBranch = "main"

registryNamespace = "k8s"
registryUrl = "registry.cloudogu.com"

helmTargetDir = "target/k8s"
helmChartDir = "${helmTargetDir}/helm"

parallel(
    "source code": {
        timestamps {
            node('docker') {
                properties([
                        // Don't run concurrent builds for a branch, because they use the same workspace directory
                        disableConcurrentBuilds(),
                ])

                stage('Checkout') {
                    checkout scm
                }

                stage('Lint') {
                    Dockerfile dockerfile = new Dockerfile(this)
                    dockerfile.lint()
                }

                stage('Check markdown links') {
                    Markdown markdown = new Markdown(this, "3.11.0")
                    markdown.check()
                }

                withGolangContainer {
                    stage('Build') {
                        sh "make vendor"
                        sh "make clean compile-ci"
                    }

                    stage('Unit Test') {
                        sh "make unit-test"
                        junit allowEmptyResults: true, testResults: 'target/unit-tests/*-tests.xml'
                    }
                    /* // golangci-linter is currently not available for go1.25
                    stage('Static Analysis') {
                        def commitSha = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()

                        withCredentials([
                                [$class: 'UsernamePasswordMultiBinding', credentialsId: 'sonarqube-gh', usernameVariable: 'USERNAME', passwordVariable: 'REVIEWDOG_GITHUB_API_TOKEN']
                        ]) {
                            withEnv(["CI_PULL_REQUEST=${env.CHANGE_ID}", "CI_COMMIT=${commitSha}", "CI_REPO_OWNER=cloudogu", "CI_REPO_NAME=${repositoryName}"]) {
                                sh "make static-analysis-ci"
                            }
                        }
                    }
                    */
                }

                stage('SonarQube') {
                    projectName = 'ces-importer'
                    def scannerHome = tool name: 'sonar-scanner', type: 'hudson.plugins.sonar.SonarRunnerInstallation'
                    withSonarQubeEnv {
                        sh "git config 'remote.origin.fetch' '+refs/heads/*:refs/remotes/origin/*'"
                        branch = env.BRANCH_NAME
                        gitWithCredentials("fetch --all")

                        if (branch == "main") {
                            echo "This branch has been detected as the main branch."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName}"
                        } else if (branch == "develop") {
                            echo "This branch has been detected as the develop branch."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=main  "
                        } else if (env.CHANGE_TARGET) {
                            echo "This branch has been detected as a pull request."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.pullrequest.key=${env.CHANGE_ID} -Dsonar.pullrequest.branch=${env.CHANGE_BRANCH} -Dsonar.pullrequest.base=develop    "
                        } else if (branch.startsWith("feature/")) {
                            echo "This branch has been detected as a feature branch."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
                        } else if (branch.startsWith("bugfix/")) {
                            echo "This branch has been detected as a bugfix branch."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
                        } else {
                            echo "This branch has been detected as a miscellaneous branch."
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=${projectName} -Dsonar.projectName=${projectName} -Dsonar.branch.name=${branch} -Dsonar.branch.target=develop"
                        }
                    }
                    timeout(time: 2, unit: 'MINUTES') { // Needed when there is no webhook for example
                        def qGate = waitForQualityGate()
                        if (qGate.status != 'OK') {
                            unstable("Pipeline unstable due to SonarQube quality gate failure")
                        }
                    }
                }

                if (gitflow.isReleaseBranch()) {
                    Makefile makefile = new Makefile(this)
                    String releaseVersion = makefile.getVersion()
                    String changelogVersion = git.getSimpleBranchName()

                    stage('Build & Push Image') {
                        def coordinatorImageName = "cloudogu/ces-importer:${releaseVersion}"
                        def coordinatorImage = docker.build("${coordinatorImageName}", "--progress=plain .")
                        docker.withRegistry('https://registry.hub.docker.com/', 'dockerHubCredentials') {
                            coordinatorImage.push()
                        }

                        def jobImageName = "cloudogu/ces-importer-migration-job:${releaseVersion}"
                        def jobImage = docker.build("${jobImageName}", "--progress=plain \
                                    --build-arg BINARY=import-job \
                                    --build-arg UID=0 \
                                    --build-arg GID=0 .")
                        docker.withRegistry('https://registry.hub.docker.com/', 'dockerHubCredentials') {
                            jobImage.push()
                        }
                    }

                    stage('Push Helm chart to Harbor') {
                        new Docker(this)
                            .image("golang:${goVersion}")
                            .mountJenkinsUser()
                            .inside("--volume ${WORKSPACE}:/${repositoryName} -w /${repositoryName}")
                                {
                                    sh "make helm-package"
                                    archiveArtifacts "${helmTargetDir}/**/*"

                                    withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'harborhelmchartpush', usernameVariable: 'HARBOR_USERNAME', passwordVariable: 'HARBOR_PASSWORD']]) {
                                        sh ".bin/helm registry login ${registryUrl} --username '${HARBOR_USERNAME}' --password '${HARBOR_PASSWORD}'"
                                        sh ".bin/helm push ${helmChartDir}/${repositoryName}-${releaseVersion}.tgz oci://${registryUrl}/${registryNamespace}"
                                    }
                                }
                    }

                    stage('Finish Release') {
                        gitflow.finishRelease(changelogVersion, productionReleaseBranch)
                    }

                    stage('Add Github-Release') {
                        github.createReleaseWithChangelog(changelogVersion, changelog, productionReleaseBranch)
                    }
                }
            }
        }
    }
)

void withGolangContainer(Closure closure) {
    new Docker(this)
            .image("golang:${goVersion}")
            .mountJenkinsUser()
            .inside("-e ENVIRONMENT=ci") {
                closure.call()
            }
}

void gitWithCredentials(String command) {
    withCredentials([usernamePassword(credentialsId: 'cesmarvin', usernameVariable: 'GIT_AUTH_USR', passwordVariable: 'GIT_AUTH_PSW')]) {
        sh(
                script: "git -c credential.helper=\"!f() { echo username='\$GIT_AUTH_USR'; echo password='\$GIT_AUTH_PSW'; }; f\" " + command,
                returnStdout: true
        )
    }
}
