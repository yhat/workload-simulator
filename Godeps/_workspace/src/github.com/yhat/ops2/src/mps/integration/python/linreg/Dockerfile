FROM continuumio/anaconda:latest

run conda install pandas==0.16.1

{{ INSTALL YHAT }}

copy model.py /root/model.py
workdir /root

cmd ["python", "model.py"]
