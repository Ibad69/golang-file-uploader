pipeline {
    agent any
    environment {
        SSH_USER = 'devxonic'
        SSH_HOST = '192.168.18.9'
        RUN_SUDO = 'export SUDO_ASKPASS=/home/devxonic/secret/mypass.sh'
        APP_NAME = "golang-file-uploader"
        REPO_NAME = "golang-file-uploader"
        REPO_URL = "git@github.com:Ibad69/golang-file-uploader.git"
        BRANCH = "main"
        DISCORD_WEBHOOK = "https://discord.com/api/webhooks/1329384928579817513/dJIdE2afGsiQtloHfcnVJNMzOmNYypvyHsp-fKPYQ9ktHLEpGTP1JRHejfJYs0zPYZqK"
        SERVICE_NAME = "golang-file-uploader.service"
    }
    stages {
        stage("Start") {
            steps {
                echo "Go-Lang Pipeline Execution Started."
            }
        }
        stage("Git Pull") {
            steps {
                sshagent(['ssh']) {
                    echo "Pulling latest code from Git repository..."
                    sh """
                        ssh -o StrictHostKeyChecking=no ${SSH_USER}@${SSH_HOST} << ENDSSH
                        set -x

                        # Check if the development directory exists
                        if [ ! -d '/home/devxonic/development' ]; then
                            echo 'Directory development does not exist. Creating it...';
                            mkdir '/home/devxonic/development';
                        else
                            echo 'Navigating to the /home/devxonic/development...';
                            cd /home/devxonic/development;
                        fi

                        # List files to ensure we're in the right directory
                        echo 'Listing contents of development directory...';
                        ls -la;

                        # Check if the repository folder exists inside development
                        if [ ! -d '${REPO_NAME}' ]; then
                            echo 'Repository folder does not exist. Cloning repository...';
                            git clone ${REPO_URL} ${REPO_NAME};
                            cd ${REPO_NAME};
                            git switch ${BRANCH};
                        else
                            echo 'Repository folder exists. Checking if it is a Git repository...';
                            cd ${REPO_NAME};
    
                            # Check if it's a Git repository
                            if [ ! -d '.git' ]; then
                                echo 'Not a Git repository. Initializing repository...';
                                git init;
                                git remote add origin ${REPO_URL};
                                git fetch origin;
                                git switch ${BRANCH};
                            else
                                echo 'Directory is a Git repository. Pulling latest changes...';
                                git fetch origin;
                                git switch ${BRANCH};
                                git pull origin ${BRANCH};
                            fi
                        fi
                    """
                }
            }
        }
        stage("Build") {
            steps {
                sshagent(['ssh']){
                    echo "Connecting to machine..."
                    sh """
                    ssh -o StrictHostKeyChecking=no ${SSH_USER}@${SSH_HOST} << ENDSSH
                        
                    export ${RUN_SUDO};
                        
                    cd /home/devxonic/development/${REPO_NAME}/cmd/api;
                        
                    ls -la;
                        
                    go version;

                    if [ -f 'main' ]; then
                      echo "Removing main file...";
                      rm main;
                    else
                      echo "Building main file...";
                      go build main.go;
                    fi
                    
                    """
                }
            }
        }
        stage("Go Service") {
            steps {
                sshagent(['ssh']){
                    echo "Connecting to machine..."
                    sh """
                    ssh -o StrictHostKeyChecking=no ${SSH_USER}@${SSH_HOST} << ENDSSH
                        
                    export ${RUN_SUDO};

                    # Check if the ${SERVICE_NAME} file exists
                    if [ ! -f /etc/systemd/system/${SERVICE_NAME} ]; then
                        echo "Creating ${SERVICE_NAME} file..."

                        # Create the systemd service file for golang-file-uploader
                        echo '
[Unit]
Description=golang-file-uploader Go Application
After=network.target
                        
[Service]
Type=simple
Restart=always
RestartSec=5s
ExecStart=/home/devxonic/development/golang-file-uploader/cmd/api/main
WorkingDirectory=/home/devxonic/development/golang-file-uploader/cmd/api
EnvironmentFile=/home/devxonic/development/golang-file-uploader/.env
                        
[Install]
WantedBy=multi-user.target
                        ' | sudo -A tee /etc/systemd/system/${SERVICE_NAME} > /dev/null

                        # Reload systemd to recognize the new service
                        sudo -A systemctl daemon-reload;

                        # Start and enable the service
                        sudo -A systemctl start ${SERVICE_NAME};
                        sudo -A systemctl enable ${SERVICE_NAME};
                    else
                        echo "${SERVICE_NAME} file already exists."
                    fi
                    
                    # Check the status of the ${SERVICE_NAME}
                    service_status=\$(sudo -A systemctl is-active ${SERVICE_NAME})
                    
                    if [ "\$service_status" == "active" ]; then
                        echo "${SERVICE_NAME} is already running."
                        
                        # Restart the service
                        echo "Restarting ${SERVICE_NAME}..."
                        sudo -A systemctl restart ${SERVICE_NAME};
                    
                    else
                        echo "${SERVICE_NAME} is not running. Starting the service..."
                        
                        # Start the service if it is not active
                        sudo -A systemctl start ${SERVICE_NAME};
                    
                    fi

                    sudo -A systemctl restart ${SERVICE_NAME};

                    sudo -A systemctl status ${SERVICE_NAME};

                    """
                }
            }
        }
        stage("End") {
            steps {
                script {
                    if (currentBuild.result == null || currentBuild.result == 'SUCCESS') {
                        echo "Pipeline completed successfully. 🎉"
                    } else {
                        echo "Pipeline encountered errors. Please check the logs. ❌"
                    }
                }
            }
        }
    }
    post {
        success {
            discordSend description: "✅ Pipeline succeeded for ${APP_NAME}!", 
                        footer: "Jenkins Pipeline Notification", 
                        link: env.BUILD_URL, 
                        result: "SUCCESS", 
                        title: env.JOB_NAME, 
                        webhookURL: env.DISCORD_WEBHOOK
        }
        failure {
            discordSend description: "❌ Pipeline failed for ${APP_NAME}. Check logs!", 
                        footer: "Jenkins Pipeline Notification", 
                        link: env.BUILD_URL, 
                        result: "FAILURE", 
                        title: env.JOB_NAME, 
                        webhookURL: env.DISCORD_WEBHOOK
        }
        aborted {
            discordSend description: "⚠️ Pipeline was **aborted** for ${APP_NAME}.", 
                        footer: "Jenkins Pipeline Notification", 
                        link: env.BUILD_URL, 
                        result: "ABORTED", 
                        title: env.JOB_NAME, 
                        webhookURL: env.DISCORD_WEBHOOK
        }
        always {
            echo "Pipeline completed."
        }
    }
}
