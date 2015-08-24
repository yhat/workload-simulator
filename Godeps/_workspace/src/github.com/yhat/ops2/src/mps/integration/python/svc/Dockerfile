FROM continuumio/anaconda:latest

{{ INSTALL YHAT }}

copy svc.py /root/svc.py
workdir /root

cmd ["python", "svc.py"]
