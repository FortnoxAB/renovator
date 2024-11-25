properties(
	[
		buildDiscarder(
			logRotator(
				numToKeepStr: '5'
			)
		)
	]
)

node('go1.23') {
	container('run'){
		def tag = ''
		stage('Checkout') {
			checkout scm
			tag = sh(script: 'git tag -l --contains HEAD', returnStdout: true).trim()
		}

		stage('Fetch dependencies') {
			// using ID because: https://issues.jenkins-ci.org/browse/JENKINS-32101
			sshagent(credentials: ['18270936-0906-4c40-a90e-bcf6661f501d']) {
				sh('go mod download')
			}
		}

		stage('Run test') {
			sh('make test')
		}

		if (env.BRANCH_NAME == 'master') {
			stage('Generate and push docker image'){
				docker.withRegistry("https://quay.io", 'docker-registry') {
					strippedTag = tag.replaceFirst('v', '')
					sh("make push VERSION=${strippedTag}")
				}
			}
		}
	}
}
