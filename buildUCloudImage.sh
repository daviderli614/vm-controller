TAG=$1
if [ ! -n "$TAG" ]; then
  echo "lack TAG"
  exit 1
fi
docker build -t uhub.service.ucloud.cn/liyang01/vm-controller:$TAG .
