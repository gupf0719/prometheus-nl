#!/usr/bin/env bash

svn up

declare -A argdic
for arg in $@
do
    arr=(${arg//=/ })
    argdic[${arr[0]#*--}]=${arr[1]}
done

registryServer=${argdic["registryServer"]}
paascluster=${argdic["paascluster"]}
version=${argdic["version"]}

echo $registryServer
echo $paascluster

#registryServer=test44:5000
#paascluster=172.32.149.60
datetimestamp=`date "+%Y%m%d%H%M"`
deployVersion=v2.27.0.${datetimestamp}
if [ "X" == "X$version" ];then
   version=${deployVersion}
fi

export GOROOT=/home/go
export PATH=/home/go/bin:$PATH
export GO111MODULE=on
go version
go env
go build --mod=mod  -o prometheus ./cmd/prometheus/main.go

if [ $? -ne 0 ]; then
     exit 1
fi

cat ./Dockerfile_tpl > ./Dockerfile
sed -i "s/\$registryServer/${registryServer}/g" ./Dockerfile
imageTag="${registryServer}/paas_public/prometheus:${version}"
docker build -t ${imageTag} .
if [ $? -ne 0 ];then
    echo "docker build failed"
    exit 1
fi

docker push ${imageTag}
if [ "$deployVersion" != "$version" ];then
   deployimageTag="${registryServer}/paas_public/prometheus::${deployVersion}"
   docker tag ${imageTag} ${deployimageTag}
   docker push ${deployimageTag}
fi


echo "[paascluster]" > ./hosts
echo ${paascluster} >> ./hosts
#ansible-playbook restart.yaml -i ./hosts --extra-vars "imageTag=${imageTag}"