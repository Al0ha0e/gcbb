## GCBB

### Usage

<b>Caution! It cannot work behind a symmetric NAT.</b>

`go run main.go <staticPort> <selfId>`

### Docker

#### Images

https://hub.docker.com/repository/docker/al0ha0e/gcbb

#### Build
- amd64 `docker build -t al0ha0e/gcbb:v1 .`

- arm64 `docker buildx build --platform linux/arm64 --load -t al0ha0e/gcbb:v1_arm64 .`

#### Run

`docker run --network=host -it al0ha0e/gcbb:<Version>`