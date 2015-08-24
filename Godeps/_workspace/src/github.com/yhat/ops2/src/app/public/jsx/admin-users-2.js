var UserEditModal = React.createClass({
  getInitialState: function() {
    return { username: '', apikey: '', isAdmin: false };
  },
  render: function() {
    return (
      <div className="modal fade" id="userModal" tabindex="-1" role="dialog" aria-labelledby="userLabel">
        <div className="modal-dialog modal-lg" role="document">
          <div className="modal-content">
            <div className="modal-header">
              <button type="button" className="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
              <h4 className="modal-title" id="userLabel">User Profile</h4>
            </div>
            <div className="modal-body">
              <div className="row">
                <div className="col-sm-4">
                  <var className="lead small text-primary pull-right">Username</var>
                </div>
                <div className="col-sm-8">
                  <p className="small text-muted">{this.state.username}</p>
                </div>
              </div>
              <hr />
              <div className="row">
                <div className="col-sm-4">
                  <var className="lead small text-primary pull-right">API Key</var>
                </div>
                <div className="col-sm-8">
                  <p className="small text-muted"><code>{this.state.apikey}</code></p>
                </div>
              </div>
              <hr />
              <div className="row">
                <div className="col-sm-4">
                  <var className="lead small text-primary pull-right">Admin</var>
                </div>
                <div className="col-sm-8">
                  <button className="btn btn-default btn-xs">Make Admin</button>
                </div>
              </div>
              <hr />
              <div className="row">
                <div className="col-sm-4">
                  <var className="lead small text-primary pull-right">Reset Password</var>
                </div>
                <div className="col-sm-6">
                  <div>
                    <div className="form-group">
                      <input ref="password" type="password" name="password" className="form-control input-sm" placeholder="New Password" required/>
                    </div>
                    <div className="form-group">
                      <input ref="password" type="password" name="password" className="form-control input-sm" placeholder="Confirm New Password" required/>
                    </div>
                    <button className="btn btn-default btn-xs">Change Password</button>
                  </div>
                </div>
              </div>
              <hr />
              <div className="row">
                <div className="col-sm-4">
                  <var className="lead small text-primary pull-right">Delete User</var>
                </div>
                <div className="col-sm-6">
                  <p className="small text-muted">Fully delete this user from the ScienceOps system.</p>
                  <p className="small text-muted"><strong><span className="text-danger">Warning: deletion is not reversible.</span></strong></p>
                  <button className="btn btn-danger btn-xs btn-block">Delete</button>
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button type="button" className="btn btn-default" data-dismiss="modal">Close</button>
            </div>
          </div>
        </div>
      </div>
      );
  }
});

var UsersTable = React.createClass({

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
          <tr>
            <td>{user.Name}</td>
            <td><span className={className}>admin</span></td>
            <td><button className="btn btn-xs btn-default" onClick={this.setModalData}>edit</button></td>
          </tr>
        );
    },
    render: function() {
        var headers = ["User", "Admin?", ""];
        return (
            <table className="table">
                <tbody>
                  {this.state.users.map(function(u) {
                    return <UserRow name={u.Name} apikey={u.Apikey} isAdmin={u.Admin} />
                  })}
                </tbody>
            </table>
        )
    }
});

var UserRow = React.createClass({
  handleClick: function() {
    $("#username").text(this.props.name);
    $("#apikey").text(this.props.apikey);
    $("#admin-checkbox").bootstrapSwitch('state', this.props.isAdmin);
    $("#userModal").modal('show');
  },
  render: function() {
      var className = !this.props.isAdmin ? "hide" : "label label-primary";
      return (
        <tr>
          <td>{this.props.name} <sup><span className={className}>admin</span></sup></td>
          <td><button className="btn btn-xs btn-default" onClick={this.handleClick}>edit</button></td>
        </tr>
      );
  }
});




var CreateUser = React.createClass({

    getInitialState: function() {
        return {error:""};
    },

    renderError: function() {
        if (this.state.error === "") {
            return <div></div>
        }
        return (
          <div className="alert alert-warning alert-dismissible" role="alert">
            <button type="button" className="close" data-dismiss="alert" aria-label="Close"><span aria-hidden="true">&times;</span></button>
            {this.state.error}
          </div>
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
        <div>
          <hr/>
          <h4>Create a New User</h4>
          <div>
            <div className="form-group">
              <label for="user">User</label>
              <input ref="username" type="text" name="username" className="form-control" 
                  id="user" placeholder="Username" />
            </div>
            <div className="form-group">
              <label for="email">Email</label>
              <input ref="email" type="email" name="email" className="form-control"
                  id="email" placeholder="Email" />
            </div>
            <div className="form-group">
              <label for="pass">Password</label>
              <input ref="password" type="password" name="password" className="form-control" 
                  id="pass" placeholder="Password" />
            </div>
            <div className="checkbox">
              <label>
                <input ref="admin" type="checkbox" /> Admin
              </label>
            </div>
            <button onClick={this.submit} className="btn btn-default">Create User</button>
          </div>
          <br />
          {this.renderError()}
        </div>
      )
    }
});

var usersTable = <UsersTable url="/admin/users.json" pollInterval={2000}/>

React.render(
    usersTable, 
    document.getElementById("users-table")
);

React.render(<AdminNav page="admin" route={window.location.pathname} />, document.getElementById("admin-nav"));