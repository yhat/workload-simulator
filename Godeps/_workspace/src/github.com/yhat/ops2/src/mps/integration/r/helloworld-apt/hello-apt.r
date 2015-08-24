library(yhatr)

model.transform  <- function(request) {
    me <- request$name
    paste ("Hello", me, "!")
}
model.predict <- function(greeting) {
    data.frame(greeting=greeting)
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
yhat.deploy("HelloWorldAptPkgR", packages=c("tree","nmap"))
