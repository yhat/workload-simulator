var CodeSamples = {
    Python: "from yhat import Yhat\nimport json\n\nyh = Yhat(\n    \"{USERNAME}\",\n    \"{APIKEY}\",\n    \"{DOMAIN}/\"\n)\ndata = json.loads('{DATA}')\nyh.predict(\"{MODEL_NAME}\", data)",
    R: "library(jsonlite)\nlibrary(yhatr)\n\nyhat.config <- c(\n    username=\"{USERNAME}\",\n    apikey=\"{APIKEY}\",\n    env=\"{DOMAIN}/\"\n)\n\ndata <- fromJSON('{DATA}')\nyhat.predict(\"{MODEL_NAME}\", data)",
    cURL: "curl -X POST -H \"Content-Type: application/json\" \\\n    --user {USERNAME}:{APIKEY} \\\n    --data '{DATA}' \\\n    {DOMAIN}/{USERNAME}/models/{MODEL_NAME}/"
}

var DeploySamples = {
	R: "library(yhatr)\n\nmodel.predict <- function(request) {\n    name <- request$name\n    greeting <- paste(\"Hello\", name)\n    data.frame(greeting=greeting)\n}\n\nyhat.config  <- c(\n    username=\"{USERNAME}\",\n    apikey=\"{APIKEY}\",\n    env=\"{DOMAIN}\"\n)\nyhat.deploy(\"HelloWorld\")",
	Python: "from yhat import Yhat, YhatModel, preprocess\n\nclass HelloWorld(YhatModel):\n    @preprocess(in_type=dict, out_type=dict)\n    def execute(self, data):\n        me = data['name']\n        greeting = \"Hello %s!\" % me\n        return { \"greeting\": greeting }\n\nyh = Yhat(\n    \"{USERNAME}\",\n    \"{APIKEY}\",\n    \"{DOMAIN}\"\n)\nyh.deploy(\"HelloWorld\", HelloWorld, globals())"
}
