var User = React.createClass({displayName: "User",
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
    var cb = React.createElement("input", {ref: "checkbox", type: "checkbox", name: "my-checkbox", onChange: handle})

    if(this.state.shared) {
        cb = React.createElement("input", {ref: "checkbox", type: "checkbox", name: "my-checkbox", onChange: handle, checked: true})
        
    }
    return (
      React.createElement("li", {className: "list-group-item"}, 
        this.props.username, 
        React.createElement("span", {className: "pull-right"}, React.createElement("span", {ref: "checkboxContainer"}, 
        cb
        ))
      )
    );
  }
});

var SharedUsers = React.createClass({displayName: "SharedUsers",  
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
      React.createElement("li", {className: "list-group-item active"}, 
        React.createElement("strong", null, "Username"), 
        React.createElement("span", {className: "pull-right"}, "Shared?")
      )
    ];
    this.state.users.forEach(function(user) {
      users.push(React.createElement(User, {user_id: user.Id, username: user.Name, shared: user.Shared}));
    }.bind(this));
    return React.createElement("ul", {className: "list-group"}, users)
  }
});


var AcitonButton = React.createClass({displayName: "AcitonButton",
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
        React.createElement("button", {onClick: this.handleClick, className: klass}, this.state.text, "...")
      );
    } else {
      return (
        React.createElement("button", {onClick: this.handleClick, className: klass}, this.state.text)
      );          
    }
  }
});

var modelname = window.location.pathname.split("/")[2];
var actionURL = ["", "models", modelname, "action"].join("/");
var sharedURL = ["", "models", modelname, "shared"].join("/");

React.render(React.createElement(SharedUsers, {url: sharedURL, pollInterval: 2000}), document.getElementById('shared-users'));

React.render(React.createElement(ModelHeader, {url: window.location.pathname.split("/").slice(0, 3).join("/") + "/json"}), document.getElementById("model-header"));
React.render(React.createElement(ModelNav, {page: "settings", route: window.location.pathname.split("/").slice(0, 3).join("/")}), document.getElementById("model-nav"));

React.render(React.createElement(AcitonButton, {text: "Restart", baseurl: actionURL}), document.getElementById("action-restart"));
React.render(React.createElement(AcitonButton, {text: "Sleep", baseurl: actionURL}), document.getElementById("action-sleep"));
React.render(React.createElement(AcitonButton, {text: "Wake", baseurl: actionURL}), document.getElementById("action-wake"));
React.render(React.createElement(AcitonButton, {text: "Delete", baseurl: actionURL}), document.getElementById("action-delete"));
