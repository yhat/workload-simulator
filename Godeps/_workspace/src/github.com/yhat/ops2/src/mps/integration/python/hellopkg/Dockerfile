from ubuntu:14.04


run apt-get update --fix-missing

run apt-get install -y python-software-properties \
                       build-essential \
                       software-properties-common

run add-apt-repository ppa:fkrull/deadsnakes

run apt-get update

run apt-get install -y python2.7 python-dev curl wget unzip

run curl https://bootstrap.pypa.io/get-pip.py | python

copy foo /root/foo
copy hello.py /root/hello.py
copy module.py /root/module.py

workdir /root

{{ INSTALL YHAT }}

cmd ["python", "hello.py"]
