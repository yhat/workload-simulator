from yhat/scienceops-r:0.0.2

run Rscript -e 'install.packages("fields", repos="http://cran.yhathq.com")'

copy model.r /root/model.r

workdir /root

run wget https://github.com/yhat/yhatr/archive/master.zip
run unzip master.zip
run Rscript -e 'install.packages("/root/yhatr-master", repos = NULL, type="source")'

cmd ["Rscript", "model.r"]
