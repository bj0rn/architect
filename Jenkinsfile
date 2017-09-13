node {

    stage 'Load shared libraries'

    stage 'Checkout'
    checkout scm


    stage 'Test og coverage'
    def root = tool name: 'Go 1.8', type: 'go'
    withEnv(["GOROOT=${root}", "PATH+GO=${root}/bin"]) {
      try {
        sh './jenkins.sh'
        step([$class: 'CoberturaPublisher', coberturaReportFile: '**/coverage.xml'])
      } catch(e) {
        currentBuild.result = 'FAILURE'
      } finally {
        step([$class: 'JUnitResultArchiver', testResults: 'TEST-junit.xml'])
      }
    }

    stage 'OpenShift build'

}


