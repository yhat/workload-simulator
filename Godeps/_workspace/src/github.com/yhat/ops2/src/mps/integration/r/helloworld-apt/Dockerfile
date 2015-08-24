from yhat/scienceops-r:0.0.2

copy hello-apt.r /root/hello-apt.r

workdir /root

run wget https://github.com/yhat/yhatr/archive/master.zip
run unzip master.zip
run Rscript -e 'install.packages("/root/yhatr-master", repos = NULL, type="source")'

cmd ["Rscript", "hello-apt.r"]
