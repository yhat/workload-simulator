import os

from yhat import Yhat, YhatModel, preprocess

class HelloWorld(YhatModel):
    version = os.environ["MODEL_VERSION"]
    @preprocess(in_type=dict, out_type=dict)
    def execute(self, data):
        me = data['name']
        greeting = "Hello %s!" % me
        print os.environ["MODEL_VERSION"]
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
yh.deploy("HelloWorldVer", HelloWorld, globals(), sure=True)
