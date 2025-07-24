make image-craned-dev
#docker login ccr.ccs.tencentyun.com --username=100042475998
docker tag gocrane/craned:latest ccr.ccs.tencentyun.com/lingxiaojin/craned:v1
docker push ccr.ccs.tencentyun.com/lingxiaojin/craned:v1