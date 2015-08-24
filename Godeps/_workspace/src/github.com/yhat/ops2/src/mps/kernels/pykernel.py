import json
import os.path
import pandas as pd
import pickle
import StringIO
import subprocess
import os
import sys
import terragon
import time
import traceback

"""
Sorry for the weird variable names throughout this script. Because the user's
code is loaded and executed within the same variable space as this code, I've
prefixed all variable names with two underscores to prevent name clashes.
"""

def mkdir_p(p):
    if os.path.isdir(p):
        return
    dir = os.path.dirname(p)
    if dir:
        mkdir_p(dir)
    os.mkdir(p)

# redirect all stdout to stderr
__stdout = sys.stdout
sys.stdout = sys.stderr

__model_name = os.environ["MODELNAME"]

print "Evaluating user model code",

# This git command will show the contents of the bundle at a certain commit
# command: git show COMMIT_SHA:bundle.json
with open(sys.argv[1], 'r') as f:
    __bundle = json.load(f)

__currdir = os.path.dirname(os.path.realpath(__file__))

print "Creating user modules"

if "modules" in __bundle:
    for __module in __bundle["modules"]:
        time.sleep(.1)
        __n = __module["name"]
        __p = os.path.join(__currdir, __module["parent_dir"])
        print "creating module %s in dir %s" % (__n, __p)
        if __module["parent_dir"]:
            print "package dir detected running mkdir -p %s" % __p
            mkdir_p(__p)
        with open(os.path.join(__p, __n), 'wb+') as __f:
            __f.write(__module["source"])

print "Checking for pip requirements"

# Rather than creating another file, I just load the code using `exec()`
if "future" in __bundle:
    print "Future imports detected:\n%s" % __bundle['future'],
    exec(__bundle["future"] + "\n" + __bundle["code"])
else:
    exec(__bundle['code'])

print "Loading user objects:",
time.sleep(.01)

# Load each object into the `__tmp` variable then reassign it
for __name,__pk in __bundle['objects'].iteritems():
    print "Loading object [%s]" % __name,
    __tmp = None
    try:
	__tmp = pickle.loads(__pk)
    except:
	__tmp = terragon.loads_from_base64(__pk)
    if __tmp is None:
	print "Could not load object %s" % __name,
	sys.exit(2)
    exec('%s = __tmp' % (__name,))

# Create an instance of the user's model as `__YhatModel`
__model_class = __bundle["className"]
exec("__YhatModel = %s()" % (__model_class,))

# These things can be big. Important to delete the bundle
del __bundle

# the kernel communicates with the parent app through stdout. this function
# open's that channel, send a message, then closes it again
def __yhat_return(__msg):
    __stdout.write(__msg)
    __stdout.flush()

__yhat_return(json.dumps({"status":"UP"}))

while True:
    __line = sys.stdin.readline()
    __yhat_id = None
    # Attempt to convert the incoming JSON to a python data structure
    try:
	__data = json.loads(__line)
	if "yhat_id" in __data:
	    __yhat_id = __data.pop("yhat_id")
	else:
	    print "Got request with no yhat_id"
	    continue

        if "heartbeat" in __data:
            __yhat_return(json.dumps({"heartbeat_response": __data["heartbeat"]}))
            continue

	if "body" in __data:
	    __body = __data.pop("body")
	else:
	    __body = None
    except Exception as e:
	print "JSON parsing error on incoming data",
	__err_msg = {
	    "error": str(e),
	    "yhat_id": __yhat_id,
	    "yhat_model": __model_name
	}
	__yhat_return(json.dumps(__err_msg))
	continue

    # Time to make a prediction
    try:
	__result = __YhatModel.execute(__body)
	if isinstance(__result, pd.DataFrame):
	    # Our "standard" method for converting data frames to json
	    __colnames = __result.columns
	    __result = __result.transpose()
	    __result = __result.to_json(orient="values", date_format="iso")
	    __result = json.loads(__result)
	    __result = dict(zip(__colnames, __result))
	__result = {
	    "result": __result,
	    "yhat_id": __yhat_id,
	    "yhat_model": __model_name
	}
	__yhat_return(json.dumps(__result))
    except Exception as e:
	__err_msg = {
	    "error": str(e),
	    "yhat_id": __yhat_id,
	    "yhat_model": __model_name
	}
	__yhat_return(json.dumps(__err_msg))
	print "Error in model code:",
	__stack_trace = StringIO.StringIO()
	traceback.print_exc(file=__stack_trace)
	print __stack_trace.getvalue().strip(),
