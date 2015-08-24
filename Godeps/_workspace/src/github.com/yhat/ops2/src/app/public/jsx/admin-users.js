var UsersTable = React.createClass({

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

        return <tr>
            <td>{user.Name}</td>
            <td><button className={className} onClick={setAdmin}>{adminText}</button></td>
            <td><button className="btn btn-xs btn-danger" onClick={deleteUser}>Delete User</button></td>
        </tr>
    },

    render: function() {
        var headers = ["User", "Admin", "Delete"];
        return (
            <div>
            <table className="table">
                <thead>
                    <tr>
                {headers.map(function(header, i) { return <th>{header}</th> })}
                    </tr>
                </thead>
                <tbody>
                {this.state.users.map(this.renderUser)}
                </tbody>
            </table>
            <hr/>
            <h4>Change Password</h4>
            <div>
              <div className="form-group">
                <label>User</label>
                <select ref="username" className="form-control">
                  {this.state.users.map(function(user) {
                    return <option value={user.Name}>{user.Name}</option>
                  })}
                </select>
              </div>
              <div className="form-group">
                <label for="pass">New Password</label>
                <input ref="password" type="password" name="password" className="form-control" 
                    id="pass" placeholder="Password" />
              </div>
              <button onClick={this.changePassword} className="btn btn-default">Change Password</button>
            </div>
            </div>
        )
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
React.render(
    <CreateUser success={usersTable.getUsers}/>, 
    document.getElementById("user-create")
);



