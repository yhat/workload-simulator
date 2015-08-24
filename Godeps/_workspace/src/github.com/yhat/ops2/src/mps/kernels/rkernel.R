#!/usr/bin/env Rscript

sink(stderr())

# Variable naming note: I'm prefixing all variables with 'yhat.' to avoid
# conflicts with user created variables

yhat.model.name <- Sys.getenv("MODELNAME", "yhatmodel")

yhat.show <- sprintf("cat %s", commandArgs(trailingOnly = TRUE)[1])

# Convert the bundle into an Rdata file and load it into memory
system(paste(yhat.show,
	     " | jq .image --raw-output | base64 --decode > model.Rdata"))

load("model.Rdata")

suppressWarnings(suppressMessages(library("jsonlite")))
suppressWarnings(suppressMessages(library("rjson")))

if ("model.require" %in% ls()){
    model.require()
}

# if we don't have a model.transform function, we're just going
# to set it to the Identity function
if (! "model.transform" %in% ls()) {
    cat("model.transform function not found. function will not be executed.\n")
    model.transform <- I
}

# Open the channel to the parent process by turning off sink, send some JSON
# then reactivate sink
yhat.return <- function(msg){
    sink()
    cat(msg)
    sink(stderr())
}

# Something bad has happend. We'd better tell the user.
yhat.return.error <- function(msg, yhat.id) {
   yhat.errmsg <- data.frame(error=msg, yhat_id=yhat.id)
   yhat.errmsg <- rjson::toJSON(yhat.errmsg)
   yhat.return(yhat.errmsg)
}

# Let the parent server know that we're ready to make predictions
yhat.msg <- '{"status": "UP"}'
yhat.return(yhat.msg)

yhat.f <- file("stdin")
open(yhat.f)
# returns string w/o leading or trailing whitespace
trim <- function (x) gsub("^\\s+|\\s+$", "", x)
while(length(yhat.line <- trim(readLines(yhat.f,n=1))) > 0) {
    # You can't use `next` from inside an error function so we'll maintain
    # a flag to enable sequential try/catch statements
    yhat.model.error <- FALSE
    is.multiargs <- FALSE

    #####################################################
    ## Attempt to convert incoming JSON to an R object ##
    #####################################################
    tryCatch({
	yhat.data <- jsonlite::fromJSON(yhat.line)
	if("yhat_id" %in% names(yhat.data)) {

        if("heartbeat" %in% names(yhat.data)) {
	        yhat.msg <- rjson::toJSON(data.frame(heartbeat_response=yhat.data$heartbeat))
	        yhat.return(yhat.msg)
            next
        }

	    yhat.id <- yhat.data$yhat_id
	    # this is how you delete an element in a list
	    yhat.data$yhat_id <- NULL
	    if("args" %in% names(yhat.data)) {
	      yhat.data.input <- yhat.data$args
	      is.multiargs <- TRUE
	    } else {
	      # use rjson instead of jsonlite for "raw data"
	      yhat.data <- rjson::fromJSON(yhat.line)
	      yhat.data.input <- yhat.data$body
	    }
	} else {
	    cat("Got request with no yhat_id\n")
	    yhat.model.error <- TRUE
	}
    },
    error=function(cond) {
	print(cond)
	yhat.return.error("Could not parse incoming JSON", "UNKNOWNID")
	yhat.model.error <<- TRUE
    })
    if (yhat.model.error) { next }

    ############################
    ## Execute model.tranform ##
    ############################
    tryCatch({
	# TODO: add in `execute`
	# if it's a list, then we'll assume that we're handling multiple args
	if (is.multiargs==TRUE) {
	  yhat.transform.result <- do.call(model.transform, yhat.data.input)
	} else {
	  yhat.transform.result <- model.transform(yhat.data.input)
	}
    },
    error=function(cond){
	# grab the source code of `model.transform` and error msg
	yhat.errmsg <- paste(c("Error in model.transform function\n",
				as.character(cond),
				"model.transform <- ",
				paste(deparse(model.transform),collapse="\n")),
			     collapse="")
	cat(yhat.errmsg)
	yhat.return.error(yhat.errmsg)
	yhat.model.error <<- TRUE
    })
    if (yhat.model.error) { next }

    ###########################
    ## Execute model.predict ##
    ###########################
    tryCatch({
	yhat.model.result <- model.predict(yhat.transform.result)
    },
    error=function(cond){
	# grab the source code of `model.predict` and error msg
	yhat.errmsg <- paste(c("Error in model.predict function\n",
				as.character(cond),
				"model.predict <- ",
				paste(deparse(model.predict),collapse="\n")),
			     collapse="")
	cat(yhat.errmsg)
	yhat.return.error(yhat.errmsg, yhat.id)
	yhat.model.error <<- TRUE
    })

    if (yhat.model.error) {
      next
    }

    ################################################################
    ## convert result of model.predict to JSON and return to user ##
    ################################################################
    tryCatch({
	yhat.one.row.df <- FALSE
	if (is.data.frame(yhat.model.result)) {
	  if (nrow(yhat.model.result) == 1) {
	    yhat.one.row.df <- TRUE
	  }
	}
	yhat.df.json <- list(
	  result=yhat.model.result,
	  yhat_id=yhat.id,
	  yhat_model=yhat.model.name
	)
	if (yhat.one.row.df) {
	  if (is.null(yhat.data$non_vectorized)) {
	    yhat.df.json[["one_row_dataframe"]] = TRUE
	  }
	}
	yhat.msg <- rjson::toJSON(yhat.df.json)
	yhat.return(yhat.msg)
    },
    error=function(cond){
	yhat.errmsg <- paste(c("Error converting model result to JSON\n",
			       as.character(cond)),
			     collapse="")
	cat(yhat.errmsg)
	yhat.return.error(yhat.errmsg, yhat.id)
	yhat.model.error <<- TRUE
    })
    if (yhat.model.error) {
      next
    }
}
