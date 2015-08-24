library(datasets)
library(yhatr)

head(iris)
fit <- lm(Petal.Width ~ Sepal.Length + Sepal.Width + Petal.Length, data=iris)

model.require <- function() {
}

model.transform <- function(data){
  data
}

model.predict <- function(data) {
  pred <- predict(fit, newdata=data.frame(data))
  result <- data.frame(pred)
  print(pred)
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

yhat.deploy("LinearModelR")

