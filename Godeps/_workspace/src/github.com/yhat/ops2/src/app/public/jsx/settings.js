var User = React.createClass({
  getInitialState: function() {
    return { shared: this.props.shared }
  },
  componentDidMount: function() {
    $.fn.bootstrapSwitch.defaults.size = "mini";
    $.fn.bootstrapSwitch.defaults.onText = "Yes";
    $.fn.bootstrapSwitch.defaults.onColor = "success";
    $.fn.bootstrapSwitch.defaults.offText ="No";
    $.fn.bootstrapSwitch.defaults.offColor = "default";
    $.fn.bootstrapSwitch.defaults.animate = false;
    var ele = $(React.findDOMNode(this.refs.checkbox))
    ele.bootstrapSwitch();
    ele.on('switchChange.bootstrapSwitch', this.handleChange);
  },
  handleChange: function(e) {
    var modelname = window.location.pathname.split("/")[2];
    var url = ["", "models", modelname, "startshare", this.props.username];
    if (this.state.shared) {
        url[3] = "stopshare";
    }

    var newState = !this.state.shared;
    url = url.join("/");
    console.log(url);
    $.ajax({
      method: 'POST',
      url: url,
      cache: false,
      success: function(data) {
        this.setState({shared: newState});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  render: function() {
    var handle = this.handleChange
    var cb = <input ref="checkbox" type="checkbox" name="my-checkbox" onChange={handle} />

    if(this.state.shared) {
        cb = <input ref="checkbox" type="checkbox" name="my-checkbox" onChange={handle} checked/>
        
    }
    return (
      <li className="list-group-item">
        {this.props.username}
        <span className="pull-right"><span ref="checkboxContainer">
        {cb}
        </span></span>
      </li>
    );
  }
});

var SharedUsers = React.createClass({  
  getInitialState: function() {
    return { users: [] }
  },
  update: function() {
    $.ajax({
      url: this.props.url,
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({ users: data });
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  componentWillMount: function(){
    this.update();
    setInterval(this.update, this.props.pollInterval);
  },
  render: function() {
    var users = [
      <li className="list-group-item active">
        <strong>Username</strong>
        <span className="pull-right">Shared?</span>
      </li>
    ];
    this.state.users.forEach(function(user) {
      users.push(<User user_id={user.Id} username={user.Name} shared={user.Shared} />);
    }.bind(this));
    return <ul className="list-group">{users}</ul>
  }
});


var AcitonButton = React.createClass({
  getInitialState: function() {
    return { text: this.props.text };
  },
  handleClick: function(e) {
    var url = this.props.baseurl + "/" + this.props.text.toLowerCase();
    e.preventDefault();
    var isDelete = this.props.text == "Delete";
    $.ajax({
      url: url,
      cache: false,
      method: 'POST',
      success: function(data) {
        console.log(data);
        if (isDelete) {
           document.location.href = "/";   
        }
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  render: function() {
    var klass = "btn btn-block";
    if (this.state.text=="Delete") {
      klass = klass + " btn-danger";
    } else {
      klass = klass + " btn-default";
    }
    if (this.state.text=="Restarting") {
      return (
        <button onClick={this.handleClick} className={klass}>{this.state.text}...</button>
      );
    } else {
      return (
        <button onClick={this.handleClick} className={klass}>{this.state.text}</button>
      );          
    }
  }
});

var modelname = window.location.pathname.split("/")[2];
var actionURL = ["", "models", modelname, "action"].join("/");
var sharedURL = ["", "models", modelname, "shared"].join("/");

React.render(<SharedUsers url={sharedURL} pollInterval={2000} />, document.getElementById('shared-users'));

React.render(<ModelHeader url={window.location.pathname.split("/").slice(0, 3).join("/") + "/json"} />, document.getElementById("model-header"));
React.render(<ModelNav page="settings" route={window.location.pathname.split("/").slice(0, 3).join("/")} />, document.getElementById("model-nav"));

React.render(<AcitonButton text="Restart" baseurl={actionURL}/>, document.getElementById("action-restart"));
React.render(<AcitonButton text="Sleep" baseurl={actionURL} />, document.getElementById("action-sleep"));
React.render(<AcitonButton text="Wake" baseurl={actionURL} />, document.getElementById("action-wake"));
React.render(<AcitonButton text="Delete" baseurl={actionURL} />, document.getElementById("action-delete"));
