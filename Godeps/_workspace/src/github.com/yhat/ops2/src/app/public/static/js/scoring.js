var Highlight = React.createClass({displayName: "Highlight",
  render: function() {
    var code = this.props.code
                .replace("{USERNAME}", this.props.username).replace("{USERNAME}", this.props.username)
                .replace("{APIKEY}", this.props.apikey)
                .replace("{MODEL_NAME}", this.props.modelName)
                .replace("{DATA}", this.props.data)
                .replace("{DOMAIN}", this.props.domain)
    var highlightStyle = {fontSize: 14}
    return (
      React.createElement("pre", null, React.createElement("code", {style: highlightStyle, className: this.props.lang}, code))
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

var Scoring = React.createClass({displayName: "Scoring",
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
        return React.createElement("textarea", {className: "form-control", style: this.styles, onChange: this.updateInput, rows: "10", value: this.state.modelInput})
        break;
      case "R":
        return React.createElement(Highlight, {lang: "R", domain: domain, code: CodeSamples.R, username: this.state.user.Name, apikey: this.state.user.Apikey, data: this.state.modelInput, modelName: this.props.modelname})
        break;
      case "Python": 
        return React.createElement(Highlight, {lang: "python", domain: domain, code: CodeSamples.Python, username: this.state.user.Name, apikey: this.state.user.Apikey, data: this.state.modelInput, modelName: this.props.modelname})
        break;
      case "cURL":
        return React.createElement(Highlight, {lang: "bash", domain: domain, code: CodeSamples.cURL, username: this.state.user.Name, apikey: this.state.user.Apikey, data: this.state.modelInput, modelName: this.props.modelname})
      default:
        return "Input here";
        break;
    }
  },

  render: function() {
    var  displayInput = this.getDisplayInput();
    var executeRow;
    if (this.state.inputType == "JSON"){
      executeRow = React.createElement("button", {type: "button", className: "btn btn-primary btn-block", onClick: this.callModel}, 
          "Execute Model"
        )
    } else {
      executeRow = React.createElement("h5", null, " Execute in ", this.state.inputType == "cURL" ? "Terminal" : this.state.inputType + " session", " ")
    }

    var inputStyle = {marginLeft: 15, marginRight: 10, marginBottom: 10}
    var selectStyle = {width: 200, display: "inline"}

    return (
      React.createElement("div", null, 
        React.createElement("div", {className: "row", idx: "select-input"}, 
          React.createElement("span", {style: inputStyle}, " Select input type: "), 
          React.createElement("select", {onChange: this.updateInputType, style: selectStyle, className: "form-control"}, 
            React.createElement("option", {value: "JSON"}, "JSON"), 
            React.createElement("option", {value: "R"}, "R"), 
            React.createElement("option", {value: "cURL"}, "cURL"), 
            React.createElement("option", {value: "Python"}, "Python")
          )
        ), 
        React.createElement("div", {className: "row"}, 
          React.createElement("div", {className: "col-md-6"}, 
            displayInput, 
            executeRow
          ), 
          React.createElement("div", {className: "col-md-6"}, 
            React.createElement("textarea", {className: "form-control", style: this.styles, onKeyDown: readOnly, rows: "10", value: this.state.modelOutput}
            )
          )
        )

      )
    );
  }
});

var modelname = window.location.pathname.split("/")[2];
React.render(React.createElement(Scoring, {modelname: modelname}), document.getElementById("scoring-form"));
React.render(React.createElement(ModelHeader, {url: window.location.pathname.split("/").slice(0, 3).join("/") + "/json"}), document.getElementById("model-header"));
React.render(React.createElement(ModelNav, {page: "scoring", route: window.location.pathname.split("/").slice(0, 3).join("/")}), document.getElementById("model-nav"));