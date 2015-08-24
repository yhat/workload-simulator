FROM continuumio/anaconda:latest

{{ INSTALL YHAT }}

copy indent.py /root/indent.py
workdir /root

cmd ["python", "indent.py"]
