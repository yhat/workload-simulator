import os
import subprocess

from yhat import Yhat, YhatModel, preprocess

class HelloWorld(YhatModel):

    # ensure the environment has "tree"
    subprocess.check_output(["tree"])

    @preprocess(in_type=dict, out_type=dict)
    def execute(self, data):
        me = data['name']
        greeting = "Hello %s!" % me
        return { "greeting": greeting }


username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("PyAptGet", HelloWorld, globals(), sure=True, packages=["tree"])
