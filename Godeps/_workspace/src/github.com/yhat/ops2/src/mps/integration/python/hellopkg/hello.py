import os

from yhat import Yhat, YhatModel, preprocess
from foo.foo import print_foo
from module import function_in_same_dir

class HelloWorld(YhatModel):
    @preprocess(in_type=dict, out_type=dict)
    def execute(self, data):
        me = data['name']
        greeting = "Hello %s!" % me
        print_foo(me)
        return { "greeting": greeting, "nine": function_in_same_dir() }

username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("HelloWorldPkg", HelloWorld, globals(), sure=True, verbose=1)
