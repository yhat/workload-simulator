FROM continuumio/anaconda:latest

run conda install -y pandas==0.15.1

{{ INSTALL YHAT }}

copy beer.py /root/beer.py
copy beer_reviews.csv /root/beer_reviews.csv
workdir /root

cmd ["python", "beer.py"]
