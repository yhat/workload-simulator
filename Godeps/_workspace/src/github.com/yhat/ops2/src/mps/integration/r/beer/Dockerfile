from yhat/scienceops-r:0.0.2

ENV R_LIBS=~/local/R_libs/

run Rscript -e 'install.packages("fields", repos="http://cran.yhathq.com")'

copy beer_reviews.csv /root/beer_reviews.csv
copy beer.r /root/beer.r

workdir /root

run wget https://github.com/yhat/yhatr/archive/master.zip
run unzip master.zip
run Rscript -e 'install.packages("/root/yhatr-master", repos = NULL, type="source")'

cmd ["Rscript", "beer.r"]
