pipeline {
  agent any

  environment {
    REGISTRY        = 'ghcr.io'
    IMAGE           = 'ghcr.io/context-alarm/contextalarm-checker'
    CONTAINER_NAME  = 'contextalarm-checker'
    REMOTE_USER     = 'deploy'
    REMOTE_HOST     = '10.10.10.121'
    APP_DIR         = '/home/deploy/context-checker'
  }

  options {
    skipDefaultCheckout(true)
    timestamps()
  }

  stages {
    stage('Checkout') {
      steps { checkout scm }
    }

    stage('Set build variables') {
      steps {
        script {
          env.TAG = sh(returnStdout: true, script: 'git rev-parse --short=7 HEAD').trim()
          if (!env.TAG?.trim()) { env.TAG = env.BUILD_NUMBER ?: 'dev' }
          env.FULL_IMAGE_TAG = "${env.IMAGE}:${env.TAG}"
          echo "Image tag: ${env.FULL_IMAGE_TAG}"
        }
      }
    }

    stage('Build image') {
      steps {
        sh 'docker build -t ${FULL_IMAGE_TAG} -t ${IMAGE}:latest .'
      }
    }

    stage('Push image to GHCR') {
      steps {
        withCredentials([usernamePassword(credentialsId: 'github-https', usernameVariable: 'GHCR_USER', passwordVariable: 'GHCR_TOKEN')]) {
          sh '''
            echo "$GHCR_TOKEN" | docker login ${REGISTRY} -u "$GHCR_USER" --password-stdin
            docker push ${FULL_IMAGE_TAG}
            docker push ${IMAGE}:latest
          '''
        }
      }
    }

    stage('Deploy to 10.10.10.121') {
      steps {
        sshagent(credentials: ['base-image-pkey']) {
          withCredentials([file(credentialsId: 'checker_env', variable: 'ENV_FILE'),
                           usernamePassword(credentialsId: 'github-https', usernameVariable: 'GHCR_USER', passwordVariable: 'GHCR_TOKEN')]) {
            sh '''
              set -eu
              REMOTE="${REMOTE_USER}@${REMOTE_HOST}"
              ssh -o StrictHostKeyChecking=no "$REMOTE" "mkdir -p ${APP_DIR}"

              # Upload .env (from Jenkins secret file)
              scp -o StrictHostKeyChecking=no "$ENV_FILE" "$REMOTE:${APP_DIR}/.env"
              ssh -o StrictHostKeyChecking=no "$REMOTE" "chmod 600 ${APP_DIR}/.env && sed -i 's/\r$//' ${APP_DIR}/.env"

              # Ensure writable log directory exists
              ssh -o StrictHostKeyChecking=no "$REMOTE" "mkdir -p ${APP_DIR}/logs && chmod 777 ${APP_DIR}/logs"

              # Pull and (re)start container
              ssh -o StrictHostKeyChecking=no "$REMOTE" "\
                set -eu; \
                echo '${GHCR_TOKEN}' | docker login ${REGISTRY} -u '${GHCR_USER}' --password-stdin; \
                docker pull ${FULL_IMAGE_TAG}; \
                docker rm -f ${CONTAINER_NAME} >/dev/null 2>&1 || true; \
                docker run -d --name ${CONTAINER_NAME} --restart unless-stopped \
                  --env-file ${APP_DIR}/.env \
                  --env LOG_FILE=/logs/alarm_checker.log \
                  -v ${APP_DIR}/.env:/app/.env:ro \
                  -v ${APP_DIR}/logs:/logs \
                  --log-opt max-size=10m --log-opt max-file=3 \
                  --entrypoint sh \
                  ${FULL_IMAGE_TAG} -c 'while true; do /app/alarm-checker; sleep 900; done' \
              "
            '''
          }
        }
      }
    }
  }

  post {
    always { sh 'docker image prune -f || true' }
  }
}