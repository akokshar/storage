pipeline {
  agent {
    kubernetes {
      containerTemplate {
        name 'docker'
        image 'ubuntu'
        ttyEnabled true
        command 'cat'
      }
    }
  }

  stages {
    stage('Test') {
      steps {
        sh 'echo Test'
      }
    }
    stage('Build') {
      steps {
        sh 'echo Build'
        container('docker') {
          sh 'hostname'
        }
      }
    }
    stage('Deploy') {
      steps {
        sh 'echo Deploy'
      }
    }
  }
}
