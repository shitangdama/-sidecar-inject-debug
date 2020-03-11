go mod download
go mod vendor

GOOS=linux go build -o ./app .
docker build -t shitangdama/sidecar-inject-server .
rm -rf shitangdama/sidecar-inject-server