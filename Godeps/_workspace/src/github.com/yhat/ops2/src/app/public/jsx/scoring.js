var Highlight = React.createClass({
  render: function() {
    var code = this.props.code
                .replace("{USERNAME}", this.props.username).replace("{USERNAME}", this.props.username)
                .replace("{APIKEY}", this.props.apikey)
                .replace("{MODEL_NAME}", this.props.modelName)
                .replace("{DATA}", this.props.data)
                .replace("{DOMAIN}", this.props.domain)
    var highlightStyle = {fontSize: 14}
    return (
      <pre><code style={highlightStyle} className={this.props.lang}>{code}</code></pre>
    );
  }
});

var readOnly = function(key) {
  key.preventDefault();
  return
}

var updateExample = function(model, data) {
  var urlPath = ["", "models", model, "example"].join("/");
  console.log(urlPath);
  $.ajax({
    method: "POST",
    url: urlPath,
    data: data,
    cache: false,
    error: function(xhr, status, err) {
      console.error(urlPath, status, err.toString(), xhr.responseText);
    }.bind(this)
  });
}

var Scoring = React.createClass({
  getInitialState: function() {
    return { user: {}, modelInput: "", modelOutput: "", inputType: "JSON" };
  },
  updateInput: function(event) {
    this.setState({ modelInput: event.target.value });
  },
  updateInputType: function(event){
    this.setState({inputType: event.target.value});
  },
  highlightCode: function() {
    $('pre code').each(function(i, block) {
      hljs.highlightBlock(block);
    });
  },
  callModel: function() {
    var urlPath = ["", this.state.user.Name, "models", this.props.modelname].join("/");
    $.ajax({
      method: "POST",
      url: urlPath,
      dataType: 'json',
      headers: {
       "Authorization": "Basic " + btoa(this.state.user.Name + ":" + this.state.user.Apikey)
      },
      data: this.state.modelInput,
      cache: false,
      success: function(data) {
        this.setState({ modelOutput: JSON.stringify(data, null, 2)});
        updateExample(this.props.modelname, this.state.modelInput)
      }.bind(this),
      error: function(xhr, status, err) {
        var data = ""; 
        try {
           data = JSON.parse(xhr.responseText);
           this.setState({ modelOutput: JSON.stringify(data, null, 2) });
           updateExample(this.props.modelname, this.state.modelInput)
        } catch (err) { }
        cb(null, err);
      }.bind(this)
    });
  },
  componentWillMount: function() {
    // grab user data to make predictions with
    $.ajax({
      url: "/user.json",
      dataType: 'json',
      cache: false,
      async: false,
      success: function(data) {
        this.setState({ user: data});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });

    var urlPath = "/models/" + this.props.modelname + "/example";

    $.ajax({
      url: urlPath,
      dataType: 'text',
      cache: false,
      success: function(data) {
        this.setState({ modelInput: data });
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  styles: {fontSize: '14px', fontFamily: 'Courier, Monaco, monospace'},
  getDisplayInput: function(){
    setTimeout(this.highlightCode, 10);
    var domain = window.location.origin;
    switch(this.state.inputType) {
      case "JSON":
        return <textarea className="form-control" style={this.styles} onChange={this.updateInput} rows="10" value={this.state.modelInput}></textarea>
        break;
      case "R":
        return <Highlight lang="R" domain={domain} code={CodeSamples.R} username={this.state.user.Name} apikey={this.state.user.Apikey} data={this.state.modelInput}  modelName={this.props.modelname} />
        break;
      case "Python": 
        return <Highlight lang="python" domain={domain} code={CodeSamples.Python} username={this.state.user.Name} apikey={this.state.user.Apikey} data={this.state.modelInput}  modelName={this.props.modelname} />
        break;
      case "cURL":
        return <Highlight lang="bash" domain={domain} code={CodeSamples.cURL} username={this.state.user.Name} apikey={this.state.user.Apikey} data={this.state.modelInput}  modelName={this.props.modelname} />
      default:
        return "Input here";
        break;
    }
  },

  render: function() {
    var  displayInput = this.getDisplayInput();
    var executeRow;
    if (this.state.inputType == "JSON"){
      executeRow = <button type="button" className="btn btn-primary btn-block" onClick={this.callModel}>
          Execute Model
        </button>
    } else {
      executeRow = <h5> Execute in {this.state.inputType == "cURL" ? "Terminal" : this.state.inputType + " session"} </h5>
    }

    var inputStyle = {marginLeft: 15, marginRight: 10, marginBottom: 10}
    var selectStyle = {width: 200, display: "inline"}

    return (
      <div>
        <div className="row" idx="select-input">
          <span style={inputStyle}> Select input type: </span>
          <select onChange={this.updateInputType} style={selectStyle} className="form-control">
            <option value="JSON">JSON</option> 
            <option value="R">R</option>
            <option value="cURL">cURL</option>
            <option value="Python">Python</option>
          </select>
        </div>
        <div className="row"> 
          <div className="col-md-6">
            {displayInput}
            {executeRow}
          </div>
          <div className="col-md-6">
            <textarea className="form-control" style={this.styles} onKeyDown={readOnly} rows="10" value={this.state.modelOutput}>
            </textarea>
          </div>
        </div>

      </div>
    );
  }
});

var modelname = window.location.pathname.split("/")[2];
React.render(<Scoring modelname={modelname}/>, document.getElementById("scoring-form"));
React.render(<ModelHeader url={window.location.pathname.split("/").slice(0, 3).join("/") + "/json"} />, document.getElementById("model-header"));
React.render(<ModelNav page="scoring" route={window.location.pathname.split("/").slice(0, 3).join("/")} />, document.getElementById("model-nav"));