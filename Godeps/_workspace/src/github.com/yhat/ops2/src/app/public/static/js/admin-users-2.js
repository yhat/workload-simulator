var UserEditModal = React.createClass({displayName: "UserEditModal",
  getInitialState: function() {
    return { username: '', apikey: '', isAdmin: false };
  },
  render: function() {
    return (
      React.createElement("div", {className: "modal fade", id: "userModal", tabindex: "-1", role: "dialog", "aria-labelledby": "userLabel"}, 
        React.createElement("div", {className: "modal-dialog modal-lg", role: "document"}, 
          React.createElement("div", {className: "modal-content"}, 
            React.createElement("div", {className: "modal-header"}, 
              React.createElement("button", {type: "button", className: "close", "data-dismiss": "modal", "aria-label": "Close"}, React.createElement("span", {"aria-hidden": "true"}, "×")), 
              React.createElement("h4", {className: "modal-title", id: "userLabel"}, "User Profile")
            ), 
            React.createElement("div", {className: "modal-body"}, 
              React.createElement("div", {className: "row"}, 
                React.createElement("div", {className: "col-sm-4"}, 
                  React.createElement("var", {className: "lead small text-primary pull-right"}, "Username")
                ), 
                React.createElement("div", {className: "col-sm-8"}, 
                  React.createElement("p", {className: "small text-muted"}, this.state.username)
                )
              ), 
              React.createElement("hr", null), 
              React.createElement("div", {className: "row"}, 
                React.createElement("div", {className: "col-sm-4"}, 
                  React.createElement("var", {className: "lead small text-primary pull-right"}, "API Key")
                ), 
                React.createElement("div", {className: "col-sm-8"}, 
                  React.createElement("p", {className: "small text-muted"}, React.createElement("code", null, this.state.apikey))
                )
              ), 
              React.createElement("hr", null), 
              React.createElement("div", {className: "row"}, 
                React.createElement("div", {className: "col-sm-4"}, 
                  React.createElement("var", {className: "lead small text-primary pull-right"}, "Admin")
                ), 
                React.createElement("div", {className: "col-sm-8"}, 
                  React.createElement("button", {className: "btn btn-default btn-xs"}, "Make Admin")
                )
              ), 
              React.createElement("hr", null), 
              React.createElement("div", {className: "row"}, 
                React.createElement("div", {className: "col-sm-4"}, 
                  React.createElement("var", {className: "lead small text-primary pull-right"}, "Reset Password")
                ), 
                React.createElement("div", {className: "col-sm-6"}, 
                  React.createElement("div", null, 
                    React.createElement("div", {className: "form-group"}, 
                      React.createElement("input", {ref: "password", type: "password", name: "password", className: "form-control input-sm", placeholder: "New Password", required: true})
                    ), 
                    React.createElement("div", {className: "form-group"}, 
                      React.createElement("input", {ref: "password", type: "password", name: "password", className: "form-control input-sm", placeholder: "Confirm New Password", required: true})
                    ), 
                    React.createElement("button", {className: "btn btn-default btn-xs"}, "Change Password")
                  )
                )
              ), 
              React.createElement("hr", null), 
              React.createElement("div", {className: "row"}, 
                React.createElement("div", {className: "col-sm-4"}, 
                  React.createElement("var", {className: "lead small text-primary pull-right"}, "Delete User")
                ), 
                React.createElement("div", {className: "col-sm-6"}, 
                  React.createElement("p", {className: "small text-muted"}, "Fully delete this user from the ScienceOps system."), 
                  React.createElement("p", {className: "small text-muted"}, React.createElement("strong", null, React.createElement("span", {className: "text-danger"}, "Warning: deletion is not reversible."))), 
                  React.createElement("button", {className: "btn btn-danger btn-xs btn-block"}, "Delete")
                )
              )
            ), 
            React.createElement("div", {className: "modal-footer"}, 
              React.createElement("button", {type: "button", className: "btn btn-default", "data-dismiss": "modal"}, "Close")
            )
          )
        )
      )
      );
  }
});

var UsersTable = React.createClass({displayName: "UsersTable",

    getInitialState: function() {
        return {users: [], modal: {} };
    },

    componentDidMount: function() {
        this.getUsers();
        setInterval(this.getUsers, this.props.pollInterval);
    },

    getUsers: function() {
        $.ajax({
            url: this.props.url,
            dataType: 'json',
            cache: false,
            success: function(data) {
                this.setState({users: data, modal: {} });
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
    },

    deleteUser: function(user) {
        var reload = this.getUsers
        $.ajax({
            method: "POST",
            url: "/admin/users/delete?username=" + encodeURIComponent(user),
            cache: false,
            success: function(data) {
                reload();
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
    },

    setAdmin: function(user, admin) {
        var url = "/admin/users/unmakeadmin"
        if (admin) {
            url = "/admin/users/makeadmin";
        }
        var reload = this.getUsers
        $.ajax({
            method: "POST",
            url: url + "?username=" + encodeURIComponent(user),
            cache: false,
            success: function(data) {
                reload();
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(url, status, err.toString(), xhr.responseText);
            }.bind(this)
        });
    },
    
    changePassword: function() {
      var data = {};
      data["username"] = $(React.findDOMNode(this.refs.username)).val();
      data["password"] = $(React.findDOMNode(this.refs.password)).val();
      $.ajax({
          method: "POST",
          url: "/admin/users/setpass",
          data: $.param(data),
          cache: false,
          success: function(data) {
             $(React.findDOMNode(this.refs.password)).val("");
          }.bind(this),
          error: function(xhr, status, err) {
              console.error(this.props.url, status, err.toString());
          }.bind(this)
      });
    },
    renderUser: function(user, i) {
        var d = this.deleteUser
        var deleteUser = function() { d(user.Name); }

        var s = this.setAdmin
        var setAdmin = function() { s(user.Name, !user.Admin); }
        var className = user.Admin ? "hide" : "label label-primary";
        var adminText = user.Admin ? "Remove Admin Privileges" : "Grant Admin Privileges";

        return (
          React.createElement("tr", null, 
            React.createElement("td", null, user.Name), 
            React.createElement("td", null, React.createElement("span", {className: className}, "admin")), 
            React.createElement("td", null, React.createElement("button", {className: "btn btn-xs btn-default", onClick: this.setModalData}, "edit"))
          )
        );
    },
    render: function() {
        var headers = ["User", "Admin?", ""];
        return (
            React.createElement("table", {className: "table"}, 
                React.createElement("tbody", null, 
                  this.state.users.map(function(u) {
                    return React.createElement(UserRow, {name: u.Name, apikey: u.Apikey, isAdmin: u.Admin})
                  })
                )
            )
        )
    }
});

var UserRow = React.createClass({displayName: "UserRow",
  handleClick: function() {
    $("#username").text(this.props.name);
    $("#apikey").text(this.props.apikey);
    $("#admin-checkbox").bootstrapSwitch('state', this.props.isAdmin);
    $("#userModal").modal('show');
  },
  render: function() {
      var className = !this.props.isAdmin ? "hide" : "label label-primary";
      return (
        React.createElement("tr", null, 
          React.createElement("td", null, this.props.name, " ", React.createElement("sup", null, React.createElement("span", {className: className}, "admin"))), 
          React.createElement("td", null, React.createElement("button", {className: "btn btn-xs btn-default", onClick: this.handleClick}, "edit"))
        )
      );
  }
});




var CreateUser = React.createClass({displayName: "CreateUser",

    getInitialState: function() {
        return {error:""};
    },

    renderError: function() {
        if (this.state.error === "") {
            return React.createElement("div", null)
        }
        return (
          React.createElement("div", {className: "alert alert-warning alert-dismissible", role: "alert"}, 
            React.createElement("button", {type: "button", className: "close", "data-dismiss": "alert", "aria-label": "Close"}, React.createElement("span", {"aria-hidden": "true"}, "×")), 
            this.state.error
          )
        )
    },

    submit: function() {
      var data = {};
      data["username"] = $(React.findDOMNode(this.refs.username)).val();
      data["password"] = $(React.findDOMNode(this.refs.password)).val();
      data["email"] = $(React.findDOMNode(this.refs.email)).val();
      if (React.findDOMNode(this.refs.admin).checked) {
        data["admin"] = "true";
      }
      $.ajax({
        type: "POST",
        url: '/admin/users/create',
        data: data,
        encode: true,
        cache: false,
        success: function(data) {
          if(this.props.success) {
            this.props.success();
          }
          $(React.findDOMNode(this.refs.username)).val("");
          $(React.findDOMNode(this.refs.password)).val("");
          $(React.findDOMNode(this.refs.email)).val("");
        }.bind(this),
        error: function(xhr, status, err) {
          this.setState({error: xhr.responseText});
        }.bind(this)
      })
      alert("submitted" + username + password+email+admin);
    },

    render: function() {
      return (
        React.createElement("div", null, 
          React.createElement("hr", null), 
          React.createElement("h4", null, "Create a New User"), 
          React.createElement("div", null, 
            React.createElement("div", {className: "form-group"}, 
              React.createElement("label", {for: "user"}, "User"), 
              React.createElement("input", {ref: "username", type: "text", name: "username", className: "form-control", 
                  id: "user", placeholder: "Username"})
            ), 
            React.createElement("div", {className: "form-group"}, 
              React.createElement("label", {for: "email"}, "Email"), 
              React.createElement("input", {ref: "email", type: "email", name: "email", className: "form-control", 
                  id: "email", placeholder: "Email"})
            ), 
            React.createElement("div", {className: "form-group"}, 
              React.createElement("label", {for: "pass"}, "Password"), 
              React.createElement("input", {ref: "password", type: "password", name: "password", className: "form-control", 
                  id: "pass", placeholder: "Password"})
            ), 
            React.createElement("div", {className: "checkbox"}, 
              React.createElement("label", null, 
                React.createElement("input", {ref: "admin", type: "checkbox"}), " Admin"
              )
            ), 
            React.createElement("button", {onClick: this.submit, className: "btn btn-default"}, "Create User")
          ), 
          React.createElement("br", null), 
          this.renderError()
        )
      )
    }
});

var usersTable = React.createElement(UsersTable, {url: "/admin/users.json", pollInterval: 2000})

React.render(
    usersTable, 
    document.getElementById("users-table")
);

React.render(React.createElement(AdminNav, {page: "admin", route: window.location.pathname}), document.getElementById("admin-nav"));