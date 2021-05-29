## GCBB

### Usage

<b>Caution! It cannot work behind a symmetric NAT.</b>

`go run main.go <staticPort> <selfId>`

### Docker

#### Images

https://hub.docker.com/repository/docker/al0ha0e/gcbb

#### Build
- amd64 `docker build -t <username>/gcbb:<version> .`

- arm64 `docker buildx build --platform linux/arm64 --load -t <username>/gcbb:<version>_arm64 .`

#### Run

`docker run --network=host -it al0ha0e/gcbb:<version>`