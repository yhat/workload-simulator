import os

import pandas as pd
from sklearn import linear_model
from sklearn import datasets

from yhat import Yhat, YhatModel, preprocess, df_to_json


iris = datasets.load_iris()
X = pd.DataFrame(iris.data[:,0:3], columns=iris.feature_names[0:3])
#    sepal length (cm)  sepal width (cm)  petal length (cm)
# 0                5.1               3.5                1.4
# 1                4.9               3.0                1.4
# 2                4.7               3.2                1.3
y = pd.DataFrame(iris.data[:,3:4], columns=iris.feature_names[3:4])
#    petal width (cm)
# 0               0.2
# 1               0.2
# 2               0.2
regr = linear_model.LinearRegression()
regr.fit(X, y)


class LinReg(YhatModel):
    @preprocess(in_type=pd.DataFrame, out_type=pd.DataFrame)
    def execute(self, data):
       prediction = regr.predict(pd.DataFrame(data))
       return prediction


username = os.environ["USERNAME"]
apikey = os.environ["APIKEY"]
endpoint = os.environ["OPS_ENDPOINT"]

print "%s:%s:%s" % (username, apikey, endpoint,)

yh = Yhat(
    username,
    apikey,
    endpoint
)
yh.deploy("LinearRegression", LinReg, globals(), sure=True)
