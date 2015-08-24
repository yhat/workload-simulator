#!/usr/bin/python

import glob, os

install_tip="""
run apt-get install -y wget unzip
run wget https://github.com/yhat/yhat-client/archive/master.zip
run unzip master.zip
run pip install -e yhat-client-master
"""

install_latest="run pip install yhat==1.3.6"

prefix = """# THIS FILE WAS GENERATED, DO NOT EDIT
"""

for dockerfile in glob.glob("python/*/Dockerfile"):
    with open(dockerfile, 'r') as f:
        content = f.read()

    latest = dockerfile + "_latest"
    tip = dockerfile + "_tip"
    print "Generating", latest
    with open(latest, 'w+') as f:
        f.write(prefix + content.replace("{{ INSTALL YHAT }}", install_latest))

    print "Generating", tip
    with open(tip, 'w+') as f:
        f.write(prefix + content.replace("{{ INSTALL YHAT }}", install_tip))
