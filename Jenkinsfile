wrappedNode(label: 'docker') {
  deleteDir()
  stage "checkout"
  checkout scm

  documentationChecker("docs")
  step([$class: 'GitHubCommitStatusSetter', contextSource: [$class: 'ManuallyEnteredCommitContextSource', context: 'docs']])
}

wrappedNode(label: 'windows') {
    deleteDir()
    checkout scm
    sh 'buildscripts/covertest.py --coverdir .cover --testopts="-race" && curl -s https://codecov.io/bash | bash'
    step([$class: 'GitHubCommitStatusSetter', contextSource: [$class: 'ManuallyEnteredCommitContextSource', context: 'windows tests']])
}