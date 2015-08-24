var LogTerminal = React.createClass({displayName: "LogTerminal",
  getInitialState: function() {
    return { logLines: [], instId: "" };
  },
  getLogsFromServer: function() {
    $.ajax({
      url: this.props.url,
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({logLines: data, instId: this.state.instId});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  componentDidMount: function() {
    this.getLogsFromServer();
    setInterval(this.getLogsFromServer, this.props.pollInterval);
  },
  setInstanceId: function(instId) {
    this.setState({logLines: this.state.logLines, instId: instId});
  },
  render: function() {
    var data = "";
    var instId = this.state.instId;
    var uniqueIds = {};
    this.state.logLines.forEach(function(line, idx, arr) {
        uniqueIds[line.InstanceId] = true;
    });

    var hasId = (instId in uniqueIds)

    uniqueIds = Object.keys(uniqueIds);
    uniqueIds.sort();
    if (uniqueIds.length > 0 && !hasId) {
        instId = uniqueIds[0];
    }

    this.state.logLines.forEach(function(line, idx, arr) {
        if(line.InstanceId == instId) {
            data = data + line.Timestamp + " "+ line.Data + "\n";
        }
    });

    var setInstanceId = this.setInstanceId;
    var renderNav = function(id, i) {
      var active = "";
      if (instId == id) {
        active = "active";
      }
      var setInst = function() { setInstanceId(id); }
      var humanIndex = i + 1;
      return (
        React.createElement("li", {role: "presentation", className: active, onClick: setInst}, 
          React.createElement("a", null, "Instance ", humanIndex)
        )
      );
    }
    return (
      React.createElement("div", null, 
        React.createElement("ul", {className: "nav nav-pills"}, 
          uniqueIds.map(renderNav)
        ), 
        React.createElement("pre", {className: "terminal"}, data)
      )
    );
  }
});

var modelname = window.location.pathname.split("/")[2];
var logURL = "/models/" + modelname + "/logs/json"

React.render(React.createElement(ModelHeader, null), document.getElementById("model-header"));
React.render(React.createElement(ModelNav, {page: "logs", route: window.location.pathname.split("/").slice(0, 3).join("/")}), document.getElementById("model-nav"));

React.render(React.createElement(LogTerminal, {url: logURL, pollInterval: "2000"}), document.getElementById('terminal'));
