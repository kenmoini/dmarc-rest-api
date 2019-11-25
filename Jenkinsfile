def version
pipeline {
    agent {
        label 'nodejs'
    }
    options {
        timeout(time: 45, unit: 'MINUTES') 
    }
    stages {
        stage('Get source') {
            steps {
                git branch: 'master', url: 'https://github.com/kenmoini/dmarc-rest-api.git'
            }
        }
        stage('Get version from package.json') {
            steps {
                script {
                    version = sh (
                        script: "cat package.json | grep version | head -1 | awk -F: '{ print \$2 }' | sed 's/[\",]//g' | tr -d '[[:space:]]'",
                        returnStdout: true
                    )
                    echo "${version}"
                }
            }
        }
        stage('Install packages') {
            steps {
                sh "npm install"
            }
        }
        stage('Build site') {
            steps {
                sh "npm run build"
            }
        }
        stage('Run tests?') {
            steps {
                sh "npm run test"
            }
        }
        stage('Create Image Builder') {
            when {
                expression {
                    openshift.withCluster() {
                        openshift.withProject() {
                            return !openshift.selector("bc", "site-demo-app-slide-deck").exists();
                        }
                    }
                }
            }
            steps {
                script {
                    openshift.withCluster() {
                        openshift.withProject() {
                            openshift.newBuild("--name=site-demo-app-slide-deck", "--image-stream=nodejs:8", "--binary=true")
                        }
                    }
                }
            }
        }
        
        stage('Build Image') {
            steps {
                sh "rm -rf oc-builds && mkdir -p oc-builds"
                sh "tar --exclude='./node_modules' --exclude='./.git' --exclude='./oc-builds' -zcf oc-builds/build.tar.gz ."
                script {
                    openshift.withCluster() {
                        openshift.withProject() {
                            openshift.selector("bc", "site-demo-app-slide-deck").startBuild("--from-archive=oc-builds/build.tar.gz", "--wait=true")
                        }
                    }
                }
            }
        }
        stage('Tag Image with current version') {
            steps {
                script {
                    openshift.withCluster() {
                        openshift.withProject() {
                            openshift.tag("site-demo-app-slide-deckTYPO:latest", "site-demo-app-slide-deck:${version}")
                        }
                    }
                }
            }
        }
        stage('Deploy Application') {
            steps {
                script {
                    openshift.withCluster() {
                        openshift.withProject() {
                            if (openshift.selector('dc', 'site-demo-app-slide-deck').exists()) {
                                openshift.selector('dc', 'site-demo-app-slide-deck').delete()
                            }
                            if (openshift.selector('svc', 'site-demo-app-slide-deck').exists()) {
                                openshift.selector('svc', 'site-demo-app-slide-deck').delete()
                            }
                            if (openshift.selector('route', 'site-demo-app-slide-deck').exists()) {
                                openshift.selector('route', 'site-demo-app-slide-deck').delete()
                            }

                            openshift.newApp("site-demo-app-slide-deck:${version}").narrow("svc").expose()
                        }
                    }
                }
            }
        }
    }
}