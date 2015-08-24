import os
from yhat import Yhat, YhatModel
from pricing import Pricing

class MarketingSearchAPI(YhatModel):
    REQUIREMENTS = [
        "pandas==0.15.2",
        "numpy"
        ]
    def execute(self, data):
        result = p.predict(data)
        return result

p = Pricing()

username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("RelayRidesPricing", MarketingSearchAPI, globals(), sure=True)
