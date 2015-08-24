var Apikey = React.createClass({displayName: "Apikey",
  getInitialState: function() {
    return { apikey: '' };
  },
  componentDidMount: function(){
    $.ajax({
      url: "/whoami",
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({ apikey: data[this.props.apikey_fieldname] });
      }.bind(this),
      error: function(xhr, status, err) {
        this.setState({ apikey: ""});
      }.bind(this)
    });
  },
  render: function() {
    return (
      React.createElement("p", null, this.props.text, ": ", React.createElement("code", null, React.createElement("span", null,  this.state.apikey)))
    );
  }
});

var UserInfo = React.createClass({displayName: "UserInfo",
  getInitialState: function(){
    return { username: '' };
  },
  componentWillMount: function(){
    $.ajax({
      url: "/whoami",
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({ username: data["username"] });
      }.bind(this),
      error: function(xhr, status, err) {
      }.bind(this)
    });    
  },
  render: function() {
    return React.createElement("span", null, React.createElement("code", null, this.state.username))
  }  
});

var RegenerateApiKey = React.createClass({displayName: "RegenerateApiKey",
  getInitialState: function() {
    return { displayed: false };
  },
  handleClick: function(evt) {
    this.setState({ displayed: "validating" });
  },
  postValidation: function() {
    var q = "";
    if (this.props.text=="read-only api key") {
      q = "?readonly=true";
    }
    $.ajax({
      url: "/apikey" + q,
      method: "POST",
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({ displayed: true });
        window.location.reload();
      }.bind(this),
      error: function(xhr, status, err) {
        // this.setState({ apikey: ""});
      }.bind(this)
    });
  },
  render: function() {
    if (this.state.displayed==true) {
      return (
        React.createElement("p", null, this.props.text, ": ", React.createElement("code", null, React.createElement(Apikey, {apikey_fieldname: this.props.apikey_fieldname})))
      );
    } else if (this.state.displayed=="validating") {
      return (
        React.createElement("span", null, 
          React.createElement("button", {onClick: this.handleClick, className: "btn btn-default text-uppercase"}, "regenerate ", this.props.text), 
          React.createElement(PasswordModal, {postValidation: this.postValidation, btnMessage: "Regenerate", message: "To regenerate your API key, enter your password:"})
        )
      );
    } else if (this.state.displayed==false) {
      return (
        React.createElement("button", {onClick: this.handleClick, className: "btn btn-default text-uppercase"}, "Regenerate ", this.props.text)
      );
    }
  }
});

var PasswordModal = React.createClass({displayName: "PasswordModal",
  getInitialState: function () {
    return { show: true };
  },
  handleSubmit: function(e) {
    e.preventDefault();
    var password = React.findDOMNode(this.refs.password).value.trim();
    $.ajax({
      url: "/verify-password",
      method: "POST",
      data: "password=" + password,
      success: function(data) {
        this.setState({ show: false });
        $(".password-modal").modal('hide');
        this.props.postValidation();
      }.bind(this),
      error: function(xhr, status, err) {
        // console.error(this.props.url, status, err.toString());
        this.setState({ show: true, status: "Invalid password!" });
      }.bind(this)
    });
  },
  render: function() {
    if (this.state.show==true) {
      setTimeout(function() { $(".password-modal").modal('show'); }, 25);
    } else {
      $(".password-modal").modal('hide');
    }
    var alert;
    if (this.state.status) {
      alert = React.createElement("div", {className: "alert alert-warning", role: "alert"}, this.state.status);
    }
    return (
      React.createElement("div", {className: "modal fade password-modal", tabIndex: "-1", role: "dialog", "aria-labelledby": "validatePasswordModal"}, 
        React.createElement("div", {className: "modal-dialog modal-md"}, 
          React.createElement("div", {className: "modal-content"}, 
            React.createElement("div", {className: "modal-header"}, 
              React.createElement("h4", {className: "text-primary text-center"}, this.props.btnMessage, " Key")
            ), 
            React.createElement("div", {className: "modal-body"}, 
              alert, 
              React.createElement("form", {onSubmit: this.handleSubmit}, 
                React.createElement("div", {className: "form-group"}, 
                  React.createElement("p", {className: "small"}, this.props.message), 
                  React.createElement("input", {type: "password", className: "form-control", id: "password", ref: "password", placeholder: "password"})
                ), 
                React.createElement("div", {className: "form-group text-right"}, 
                  React.createElement("button", {type: "button", className: "btn btn-default", "data-dismiss": "modal"}, "Cancel"), 
                  " ", 
                  React.createElement("button", {type: "submit", className: "btn btn-primary"}, this.props.btnMessage)
                )
              )
            )
          )
        )
      )
    );
  }
});

//show username
React.render(React.createElement(UserInfo, null), document.getElementById('username'));
// show current APIKEY

React.render(React.createElement(Apikey, {apikey_fieldname: "apikey", text: "API KEY"}), document.getElementById('api-key'));
React.render(React.createElement(Apikey, {apikey_fieldname: "read_only_apikey", text: "READ-ONLY API KEY"}), document.getElementById('read-only-key'));
// regenerates
React.render(React.createElement(RegenerateApiKey, {apikey_fieldname: "apikey", text: "api key"}), document.getElementById('api-key-regenerate'));
React.render(React.createElement(RegenerateApiKey, {apikey_fieldname: "read_only_apikey", text: "read-only api key"}), document.getElementById('read-only-key-regenerate'));

React.render(React.createElement(SideNav, {page: "account"}), document.getElementById('side-nav'));