library(RCurl)
library(yhatr)
library(fields)
library(yhatr)

csv_url <- "https://raw.githubusercontent.com/bcaffo/courses/master/09_DevelopingDataProducts/yhat/annual_all_2013.csv"
d <- read.csv(text = getURL(csv_url), nrow = 68210)

sub <- subset(d, Parameter.Name %in% c("PM2.5 - Local Conditions", "Ozone")
              & Pullutant.Standard %in% c("Ozone 8-Hour 2008", "PM25 Annual 2006"),
              c(Longitude, Latitude, Parameter.Name, Arithmetic.Mean))

pollavg <- aggregate(sub[, "Arithmetic.Mean"],
                     sub[, c("Longitude", "Latitude", "Parameter.Name")],
                     mean, na.rm = TRUE)
pollavg$Parameter.Name <- factor(pollavg$Parameter.Name, labels = c("ozone", "pm25"))
names(pollavg)[4] <- "level"

## Remove unneeded objects
rm(d, sub)

## Write function
monitors <- data.matrix(pollavg[, c("Longitude", "Latitude")])

pollutant <- function(df) {
        x <- data.matrix(df[, c("lon", "lat")])
        r <- df$radius
        d <- rdist.earth(monitors, x)
        use <- lapply(seq_len(ncol(d)), function(i) {
                which(d[, i] < r[i])
        })
        levels <- sapply(use, function(idx) {
                with(pollavg[idx, ], tapply(level, Parameter.Name, mean))
        })
        dlevel <- as.data.frame(t(levels))
        data.frame(df, dlevel)
}

model.require <- function() {
  library(fields)
}

model.transform <- function(df) {
  df
}

model.predict <- function(df) {
  pollutant(df)
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

yhat.deploy("pollutant")