library(datasets)
library(yhatr)

#import the svm package
library(e1071)

fit <- svm(Species ~ Petal.Width + Sepal.Length + Sepal.Width + Petal.Length, data=iris)

model.require <- function() {
  library(e1071)
}

model.transform <- function(data){
  data
}

model.predict <- function(data) {
  pred <- predict(fit, newdata=data.frame(data))
  result <- data.frame(pred)
  result
}

username <- Sys.getenv("USERNAME")
print(username)
apikey <- Sys.getenv("APIKEY")
print(apikey)
endpoint <- Sys.getenv("OPS_ENDPOINT")
print(endpoint)

yhat.config  <- c(
    username=username,
    apikey=apikey,
    env=endpoint
)

yhat.deploy("SupportVectorClassifierR")
