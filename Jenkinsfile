pipeline {
    // Run the pipeline on the NTS specific agent
    agent { label 'nts' }

    environment {
        GITHUB_PROJECT='marrowc'
        GITHUB_REPO='rv'
    }

    stages {
        stage('Prep workspace') {
            steps {
                withPythonEnv('python3') {
                    sh 'pip install --upgrade pip'
                    sh 'pip install -r requirements.txt'
                }
            }
        }

        stage('Package') {
            steps {
                withPythonEnv('python3') {
                    sh 'python setup.py sdist'
                }
                archiveArtifacts(artifacts: "dist/*", followSymlinks: false)
            }
        }

        stage('Publish to PyPI') {
            when {  // Only "deploy" if on the 'main' branch
                expression { return env.BRANCH_NAME == 'main' }
            }
            steps {
                // Finalize the PyPI deployment!
                withPythonEnv('python3') {
                    withCredentials([usernamePassword(credentialsId: 'nts_pypi', passwordVariable: 'PASS', usernameVariable: 'USER')]) {
                        sh 'pip install twine'
                        sh 'twine upload dist/* -u \$USER -p \$PASS --verbose'
                    }
                }
            }
        }
    }
}
