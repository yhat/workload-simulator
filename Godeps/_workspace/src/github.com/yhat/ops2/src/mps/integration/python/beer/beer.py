import os

import pandas as pd
import numpy as np
from sklearn.metrics.pairwise import cosine_similarity
import pprint as pp

from yhat import Yhat, YhatModel, preprocess

filename = os.path.join("beer_reviews.csv")
df = pd.read_csv(filename)
# let's limit things to the top 250
n = 250
top_n = df.beer_name.value_counts().index[:n]
df = df[df.beer_name.isin(top_n)]

print "melting..."
df_wide = pd.pivot_table(df, values=["review_overall"],
                         rows=["beer_name", "review_profilename"],
                         aggfunc=np.mean).unstack()

# any cells that are missing data (i.e. a user didn't buy a particular product)
# we're going to set to 0
df_wide = df_wide.fillna(0)

# this is the key. we're going to use cosine_similarity from scikit-learn
# to compute the distance between all beers
print "calculating similarity"
dists = cosine_similarity(df_wide)

# stuff the distance matrix into a dataframe so it's easier to operate on
dists = pd.DataFrame(dists, columns=df_wide.index)

# give the indicies (equivalent to rownames in R) the name of the product id
dists.index = dists.columns

def get_sims(products):
    """
    get_top10 takes a distance matrix an a productid (assumed to be integer)
    and will calculate the 10 most similar products to product based on the
    distance matrix

    dists - a distance matrix
    product - a product id (integer)
    """
    p = dists[products].apply(lambda row: np.sum(row), axis=1)
    p = p.order(ascending=False)
    return p.index[p.index.isin(products)==False]


class BeerRecommender(YhatModel):
    @preprocess(in_type=dict, out_type=dict)
    def execute(self, data):
        # handle uft8 beer names
        beers = [beer.encode('utf8') for beer in data.get("beers", [])]

        suggested_beers = get_sims(beers)
        result = []
        for beer in suggested_beers:
            result.append({"beer": beer})
        return result

username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("BeerRecommender", BeerRecommender, globals(), sure=True)
