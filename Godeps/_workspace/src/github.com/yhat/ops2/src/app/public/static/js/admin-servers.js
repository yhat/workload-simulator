var DeleteButton = React.createClass({displayName: "DeleteButton",
  delete: function() {
    $.ajax({
      type: "POST",
      data: "id=" + String(this.props.id),
      url: "/admin/servers/remove",
      success: function(data) {
        this.props.success();
      }.bind(this),
      error: function(xhr, status, err) {
        showError("Error", xhr.responseText);
      }.bind(this)
    });
  },
  render: function() {
    return React.createElement("button", {className: "btn btn-xs btn-danger", onClick: this.delete}, 
      "Remove Server"
    )
  }
});

var ServersList = React.createClass({displayName: "ServersList",
  getInitialState: function() { return { servers: [] }; },
  componentDidMount: function() {
    this.getServers();
    setInterval(this.getServers, this.props.pollInterval);
  },
  getServers: function() {
    $.ajax({
      url: this.props.url,
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({servers: data});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  render: function() {
    var headers = ["Hostname", "Status", "Remove"];
    var renderHeader = function(header, i) { return React.createElement("th", null, header); }
    var msg = React.createElement("div", null);
    if (this.state.servers.length === 0) {
      msg = React.createElement("p", null, "No servers in rotation. Add one below.");
    }

    var refresh = this.getServers;
    var renderServer = function(server, i) {
      return React.createElement("tr", null, 
        React.createElement("td", null, server.Host), 
        React.createElement("td", null), 
        React.createElement("td", null, React.createElement(DeleteButton, {id: server.Id, success: refresh}))
      )
    }
    return (
      React.createElement("div", null, 
      React.createElement("table", {className: "table"}, 
        React.createElement("thead", null, 
          React.createElement("tr", null, headers.map(renderHeader))
        ), 
        React.createElement("tbody", null, 
          this.state.servers.map(renderServer)
        )
      ), 
      msg
      )
    )
  }
});

var serverList = React.render(
  React.createElement(ServersList, {url: "/admin/servers.json", pollInterval: 2000}),
  document.getElementById("server-list")
);

var AddServerForm = React.createClass({displayName: "AddServerForm",
  handleSumbit: function(e) {
    e.preventDefault();
    var data = $(React.findDOMNode(this.refs.form)).serialize();
    $.ajax({
      type: "POST",
      url: this.props.url,
      data: data,
      success: function(data) {
        this.props.success();
      }.bind(this),
      error: function(xhr, status, err) {
        showError("Error", xhr.responseText);
      }.bind(this)
    });
  },
  render: function() {
    return (
      React.createElement("form", {className: "form", ref: "form", onSubmit: this.handleSumbit}, 
        React.createElement("div", {className: "form-group"}, 
          React.createElement("label", {for: "host"}, "Host and Port"), 
          React.createElement("input", {type: "text", className: "form-control", name: "host", placeholder: "10.0.0.1:433"})
        ), 
        React.createElement("button", {type: "submit", className: "btn btn-primary"}, "Add Server")
      )
    )
  }
});

React.render(
  React.createElement(AddServerForm, {url: "/admin/servers", success: serverList.getServers}),
  document.getElementById("server-add")
)
React.render(React.createElement(AdminNav, {page: "admin", route: window.location.pathname}), document.getElementById("admin-nav"));