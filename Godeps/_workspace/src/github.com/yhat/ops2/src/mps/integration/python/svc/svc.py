import os

import numpy as np
import pandas as pd
from sklearn.svm import SVC
from sklearn.datasets import load_iris

from yhat import Yhat, YhatModel, preprocess, df_to_json


iris = load_iris()

X = pd.DataFrame(iris.data, columns=iris.feature_names)
y = pd.DataFrame(iris.target, columns=["flower_types"])

clf = SVC()
clf.fit(X, y["flower_types"])


class MySVC(YhatModel):
    @preprocess(in_type=pd.DataFrame, out_type=pd.DataFrame)
    def execute(self, data):
        prediction = clf.predict(pd.DataFrame(data))
        species = ['setosa', 'versicolor', 'virginica']
        result = [species[i] for i in prediction]
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
yh.deploy("SupportVectorClassifier", MySVC, globals(), sure=True)
