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
                withPythonEnv('python3') {
                    dir('python-client') {
                        sh 'pip install --upgrade pip'
                        sh 'pip install -r requirements.txt'
                    }
                }
            }
        }

        stage('Test') {
            steps {
                withPythonEnv('python3') {
                    dir('python-client') {
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
                withPythonEnv('python3') {
                    dir('python-client') {
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
                    changeset 'python-client/'
                }
            }
            steps {
                withPythonEnv('python3') {
                    dir('python-client') {
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
