import os

from yhat import Yhat, YhatModel, preprocess

class HelloWorld(YhatModel):

    VERSION = int(os.environ["MODEL_VERSION"])

    @preprocess(in_type=dict, out_type=dict)
    def execute(self, data):
        return { "version": self.VERSION }


username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("modelenvvars", HelloWorld, globals(), sure=True)
