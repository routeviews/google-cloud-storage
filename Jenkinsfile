def testReportFile = 'test_report.xml'

pipeline {
    // Run the pipeline on the NTS specific agent
    agent { label 'nts' }

    environment {
        GITHUB_PROJECT = 'routeviews'
        GITHUB_REPO = 'google-cloud-storage'
    }

    stages {
        stage('Prep workspace') {
            steps {
                dir('python-client') {
                    withPythonEnv('python3') {
                        sh 'pip install --upgrade pip'
                        sh 'pip install -r requirements.txt'
                        sh 'pip install --upgrade -e .'
                    }
                }
            }
        }

        stage('Test') {
            steps {
                dir('python-client') {
                    withPythonEnv('python3') {
                        sh "pytest --junitxml=${testReportFile} || true"
                        junit(
                            keepLongStdio: true,
                            healthScaleFactor: 100.0,
                            testResults: testReportFile
                        )
                    }
                }
            }
        }

        stage('Package') {
            steps {
                dir('python-client') {
                    withPythonEnv('python3') {
                        sh 'python setup.py sdist'
                    }
                    archiveArtifacts(artifacts: 'dist/*', followSymlinks: false)
                }
            }
        }

        stage('Publish to PyPI') {
            when {
                allOf { // Only deploy main branch, and only if the python-client directory has been updated
                    branch 'main'
                    changeset 'python-client/**'
                }
            }
            steps {
                dir('python-client') {
                    withPythonEnv('python3') {
                        withCredentials([usernamePassword(
                            credentialsId: 'nts_pypi',
                            passwordVariable: 'PASS',
                            usernameVariable: 'USER'
                        )]) {
                            sh 'pip install twine'
                            sh 'twine upload dist/* -u \$USER -p \$PASS --verbose'
                        }
                    }
                }
            }
        }
    }
}
