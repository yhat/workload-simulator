library(reshape2)
library(plyr)
library(yhatr)

df <- read.csv("./beer_reviews.csv")
n <- 2
beer_counts <- table(df$beer_name)
beers <- names(beer_counts[beer_counts > n])
df <- df[df$beer_name %in% beers,]

df.wide <- dcast(df, beer_name ~ review_profilename,
            value.var='review_overall', mean, fill=0)

dists <- dist(df.wide[,-1], method="euclidean")
dists <- as.data.frame(as.matrix(dists))
colnames(dists) <- df.wide$beer_name
dists$beer_name <- df.wide$beer_name

getSimilarBeers <- function(beers_i_like) {
  beers_i_like <- as.character(beers_i_like)
  cols <- c("beer_name", beers_i_like)
  best.beers <- dists[,cols]
  if (ncol(best.beers) > 2) {
    best.beers <- data.frame(beer_name=best.beers$beer_name, V1=rowSums(best.beers[,-1]))
  }
  results <- best.beers[order(best.beers[,-1]),]
  names(results) <- c("beer_name", "similarity")
  results[! results$beer_name %in% beers_i_like,]
}

# Note that we're not doing any tranformations
model.transform <- function(df){
  df
}

model.predict <- function(df) {
  getSimilarBeers(df$beers)
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

yhat.deploy("BeerRecommenderR")
