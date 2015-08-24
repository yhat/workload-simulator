library(yhatr)

model.transform  <- function(data) {
    data
}
model.predict <- function(data) {
    data
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
yhat.deploy("REcho")
