var UsersTable = React.createClass({displayName: "UsersTable",

    getInitialState: function() {
        return {users: []};
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
              console.log(data);
                this.setState({users: data});
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
        var className = user.Admin ? "btn btn-xs btn-warning" : "btn btn-xs btn-primary";
        var adminText = user.Admin ? "Remove Admin Privileges" : "Grant Admin Privileges";

        return React.createElement("tr", null, 
            React.createElement("td", null, user.Name), 
            React.createElement("td", null, React.createElement("button", {className: className, onClick: setAdmin}, adminText)), 
            React.createElement("td", null, React.createElement("button", {className: "btn btn-xs btn-danger", onClick: deleteUser}, "Delete User"))
        )
    },

    render: function() {
        var headers = ["User", "Admin", "Delete"];
        return (
            React.createElement("div", null, 
            React.createElement("table", {className: "table"}, 
                React.createElement("thead", null, 
                    React.createElement("tr", null, 
                headers.map(function(header, i) { return React.createElement("th", null, header) })
                    )
                ), 
                React.createElement("tbody", null, 
                this.state.users.map(this.renderUser)
                )
            ), 
            React.createElement("hr", null), 
            React.createElement("h4", null, "Change Password"), 
            React.createElement("div", null, 
              React.createElement("div", {className: "form-group"}, 
                React.createElement("label", null, "User"), 
                React.createElement("select", {ref: "username", className: "form-control"}, 
                  this.state.users.map(function(user) {
                    return React.createElement("option", {value: user.Name}, user.Name)
                  })
                )
              ), 
              React.createElement("div", {className: "form-group"}, 
                React.createElement("label", {for: "pass"}, "New Password"), 
                React.createElement("input", {ref: "password", type: "password", name: "password", className: "form-control", 
                    id: "pass", placeholder: "Password"})
              ), 
              React.createElement("button", {onClick: this.changePassword, className: "btn btn-default"}, "Change Password")
            )
            )
        )
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
React.render(
    React.createElement(CreateUser, {success: usersTable.getUsers}), 
    document.getElementById("user-create")
);



