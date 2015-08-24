var DeleteButton = React.createClass({
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
    return <button className="btn btn-xs btn-danger" onClick={this.delete}>
      Remove Server
    </button>
  }
});

var ServersList = React.createClass({
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
    var renderHeader = function(header, i) { return <th>{header}</th>; }
    var msg = <div></div>;
    if (this.state.servers.length === 0) {
      msg = <p>No servers in rotation. Add one below.</p>;
    }

    var refresh = this.getServers;
    var renderServer = function(server, i) {
      return <tr>
        <td>{server.Host}</td>
        <td></td>
        <td><DeleteButton id={server.Id} success={refresh} /></td>
      </tr>
    }
    return (
      <div>
      <table className="table">
        <thead>
          <tr>{headers.map(renderHeader)}</tr>
        </thead>
        <tbody>
          {this.state.servers.map(renderServer)}
        </tbody>
      </table>
      {msg}
      </div>
    )
  }
});

var serverList = React.render(
  <ServersList url="/admin/servers.json" pollInterval={2000}/>,
  document.getElementById("server-list")
);

var AddServerForm = React.createClass({
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
      <form className="form" ref='form' onSubmit={this.handleSumbit}>
        <div className="form-group">
          <label for="host">Host and Port</label>
          <input type="text" className="form-control" name="host" placeholder="10.0.0.1:433" />
        </div>
        <button type="submit" className="btn btn-primary">Add Server</button>
      </form>
    )
  }
});

React.render(
  <AddServerForm url="/admin/servers" success={serverList.getServers} />,
  document.getElementById("server-add")
)
React.render(<AdminNav page="admin" route={window.location.pathname} />, document.getElementById("admin-nav"));