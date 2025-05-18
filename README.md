# go-email-phishing-tools
- build docker image
```shell
docker build -t tonysarath/email-tools:latest .
```

- prepare in running environment
```shell
sudo mkdir /var/app_data
sudo chmod -R 777 /var/app_data
```
- copy `docker.env` into above `/var/app_data/`
```shell
cp docker.env targets.csv /var/app_data/
```
- copy email template (html format) into above `/var/app_data/`
- update `docker.env` accordingly
- run the docker (this will pull image from docker hub)
```shell
docker run -d \
  --name phishing-server \
  --restart unless-stopped \
  --env-file /var/app_data/docker.env \
  -p 80:8080 \
  -v "/var/app_data:/app/data" \
  tonysarath/email-tools:latest \
  serve
```

- run import via container
```shell
docker run --rm -it \
  --env-file ./docker.env \
  -v "$(pwd)/targets.csv:/app/targets.csv" \
  -v "/var/app_data:/app/data" \
  tonysarath/email-tools:latest \
  import /app/targets.csv
```
- run send via container
```shell
docker run --rm -it \
  --env-file /var/app_data/docker.env \
  -v "/var/app_data:/app/data" \
  tonysarath/email-tools:latest \
  send
```


- login to running container
```shell
docker exec -it phishing-server bash
```
- import target
```shell
./email-phishing-tools import data/targets.csv
```
---
- at the host level is we need to access database, we need to use `sqlite3` command-line
- install `sqlite3` for debain
```shell
sudo apt-get install -y sqlite3
```
- query
```shell
sqlite3 {path_to_db}/phishing_simulation.db
SQLite version 3.xx.x YYYY-MM-DD HH:MM:SS
Enter ".help" for usage hints.
sqlite>
```
- To see all tables:
```shell
.tables
```
- To see the schema of the `targets` table:
```shell
.schema targets
```
- To select all data from the `targets` table:
```shell
SELECT * FROM targets;
```
---

I'll walk you through installing Docker and Docker Compose on Ubuntu 22.04.5 LTS:

## Install Docker

First, let's set up the Docker repository:

```bash
# Update the apt package index
sudo apt-get update

# Install packages to allow apt to use a repository over HTTPS
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg

# Add Docker's official GPG key
sudo mkdir -m 0755 -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up the repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

Now install Docker Engine:

```bash
# Update the apt package index again
sudo apt-get update

# Install Docker Engine, containerd, and Docker CLI
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

Verify the installation:

```bash
# Verify Docker is installed correctly by running the hello-world image
sudo docker run hello-world
```

## Install Docker Compose

On Ubuntu 22.04, we can install Docker Compose via the Docker Compose plugin which is included in the docker-compose-plugin package we installed above. This provides the `docker compose` command:

```bash
# Check Docker Compose version
docker compose version
```

If you prefer the standalone `docker-compose` command, you can install it this way:

```bash
# Download the current stable release of Docker Compose
DOCKER_COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
sudo curl -L "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# Apply executable permissions
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker-compose --version
```

## Post-installation steps

Add your user to the docker group to run Docker without sudo:

```bash
# Add your user to the docker group
sudo usermod -aG docker $USER

# Apply the new group (you'll need to log out and back in for this to take full effect)
newgrp docker

# Verify you can run Docker commands without sudo
docker run hello-world
```

Enable Docker to start on boot:

```bash
sudo systemctl enable docker.service
sudo systemctl enable containerd.service
```
