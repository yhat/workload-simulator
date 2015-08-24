from yhat/scienceops-r:0.0.2

copy svc.r /root/svc.r

workdir /root

run wget https://github.com/yhat/yhatr/archive/master.zip
run unzip master.zip
run Rscript -e 'install.packages("/root/yhatr-master", repos = NULL, type="source")'

cmd ["Rscript", "svc.r"]
