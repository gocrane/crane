set -xv
set -e

source ~/.zshrc

GIT_VERSION=$(git describe --tags --always)
make image-crane-agent && make push-image-crane-agent
pathstr="{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"crane-agent\",\"image\":\"docker.io/gocrane/crane-agent:$GIT_VERSION\"}]}}}}"

echo "$pathstr"
kubectl -ncrane-system patch ds crane-agent  -p "$pathstr"
kdelete
kg
