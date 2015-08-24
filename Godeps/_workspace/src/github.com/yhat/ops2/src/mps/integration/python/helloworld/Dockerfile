from ubuntu:14.04


run apt-get update --fix-missing

run apt-get install -y python-software-properties \
                       build-essential \
                       software-properties-common

run add-apt-repository ppa:fkrull/deadsnakes

run apt-get update --fix-missing

run apt-get install -y python2.7 python-dev curl 

run curl https://bootstrap.pypa.io/get-pip.py | python

{{ INSTALL YHAT }}

copy hello.py /root/hello.py
workdir /root

cmd ["python", "hello.py"]
