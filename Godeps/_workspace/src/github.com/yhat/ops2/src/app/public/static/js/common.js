var ModelStatus = React.createClass({displayName: "ModelStatus",
  render: function() {
    var label_type;
    if (this.props.value=="online") {
      label_type = "label label-success";
    } else if (this.props.value=="down") {
      label_type = "label label-danger";
    } else if (this.props.value=="queued") {
      label_type = "label label-info";
    } else if (this.props.value=="building") {
      label_type = "label label-primary";
    } else if (this.props.value=="failed") {
      label_type = "label label-warning";
    } else if (this.props.value=="asleep") {
      label_type = "label label-default";
    } else {
      label_type = "";
    }
    return (
        React.createElement("span", {className: label_type}, "build ", this.props.value)
    );
  }
});

var ModelHeader = React.createClass({displayName: "ModelHeader",
  getInitialState: function() {
    return { model: {}, version: {} };
  },
  getModelFromServer: function() {
    var modelname = window.location.pathname.split("/")[2];
    var dataURL = "/models/" + modelname + "/json"
    $.ajax({
      url: dataURL,
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({ model: data.Model, version: data.Version });
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(dataURLl, status, err.toString());
      }.bind(this)
    });
  },
  componentDidMount: function(){
    this.getModelFromServer();
    setInterval(this.getModelFromServer, 1000);
  },
  render: function() {
    var img;
    if (this.state.version.Lang == "python2") {
      img = React.createElement("img", {src: "/img/logo-python.png", height: "38px"})
    } else if (this.state.version.Lang == "r") {
      img = React.createElement("img", {src: "/img/logo-r.png", height: "38px"})      
    } else {
      img = React.createElement("div", {style: {height: 38}})
    }
    return (
      React.createElement("div", null, 
        React.createElement("div", {className: "col-sm-2"}, 
          React.createElement("p", {className: "text-muted"}, React.createElement("a", {href: "/"}, React.createElement("i", {className: "fa fa-angle-left"}), " Models"))
        ), 
        React.createElement("div", {className: "col-sm-8"}, 
          React.createElement("h3", {className: "text-center"}, 
             img, " ", this.state.model.Name, " ", React.createElement("small", null, "Version ", this.state.model.ActiveVersion)
          ), 
          React.createElement("p", {className: "text-center"}, 
            React.createElement(ModelStatus, {value: this.state.model.Status})
          )
        )
      )
    );
  }
});

var ModelNav = React.createClass({displayName: "ModelNav",
  render: function() {
    return (
      React.createElement("ul", {className: "nav nav-pills nav-justified"}, 
        React.createElement("li", {className: this.props.page=="scoring" && "active", role: "presentation"}, 
          React.createElement("a", {href: this.props.route + "/scoring"}, React.createElement("i", {className: "fa fa-crosshairs"}), " Scoring")
        ), 
        React.createElement("li", {className: this.props.page=="versions" && "active", role: "presentation"}, 
          React.createElement("a", {href: this.props.route + "/versions"}, React.createElement("i", {className: "fa fa-list-ol"}), " Versions")
        ), 
        React.createElement("li", {className: this.props.page=="logs" && "active", role: "presentation"}, 
          React.createElement("a", {href: this.props.route + "/logs"}, React.createElement("i", {className: "fa fa-terminal"}), " Logs")
        ), 
        React.createElement("li", {className: this.props.page=="settings" && "active", role: "presentation"}, 
          React.createElement("a", {href: this.props.route + "/settings"}, React.createElement("i", {className: "fa fa-cogs"}), " Settings")
        )
      )
    );
  }
});

var AdminNav = React.createClass({displayName: "AdminNav",
  render: function() {
    return (
      React.createElement("ul", {className: "nav nav-pills nav-justified"}, 
        React.createElement("li", {className: location.pathname=="/admin/users" && "active", role: "presentation"}, 
          React.createElement("a", {href: "/admin/users"}, React.createElement("i", {className: "fa fa-users"}), " Users")
        ), 
        React.createElement("li", {className: location.pathname=="/admin/models" && "active", role: "presentation"}, 
          React.createElement("a", {href: "/admin/models"}, React.createElement("i", {className: "fa fa-list"}), " Models")
        ), 
        React.createElement("li", {className: location.pathname=="/admin/servers" && "active", role: "presentation"}, 
          React.createElement("a", {href: "/admin/servers"}, React.createElement("i", {className: "fa fa-server"}), " System")
        )
      )
    );
  }
})

var FormattedDate = React.createClass({displayName: "FormattedDate",

  render: function() {
    var date = new Date(this.props.value);
    date = date.toLocaleDateString() + " " + date.toLocaleTimeString();
    return (
      React.createElement("span", null, date)
    );
  }
});

var If = React.createClass({displayName: "If",
    render: function() {
        if (this.props.cond) {
            return this.props.children;
        }
        else {
            return false;
        }
    }
});

var SideNav = React.createClass({displayName: "SideNav",
  getInitialState: function() {
      return {user:{}};
  },
  componentWillMount: function(){ 
    var dataURL = "/user.json"
    $.ajax({
      url: dataURL,
      dataType: 'json',
      cache: false,
      async: false,
      success: function(data) {
        this.setState({ user: data });
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(dataURLl, status, err.toString());
      }.bind(this)
    });    
  },
  render: function() {
    var styles = {
      "paddingBottom": "5px"
    }
    var adminNav;
    if (this.state.user.Admin==true) {
      adminNav = React.createElement("li", {className: (this.state.user.Admin ? "" : " hide")}, React.createElement("a", {href: "/admin/users"}, "Admin"))
    }
    if (this.props.page=="admin" && this.state.user.Admin) {
      adminNav = (
          React.createElement("li", {className: (this.props.page=="admin" && "active")}, React.createElement("a", {href: "/admin/users"}, "Admin"), 
            React.createElement("ul", {className: "sub-nav"}, 
              React.createElement("li", {className: (location.pathname=="/admin/users" && "active")}, React.createElement("a", {href: "/admin/users"}, "Users")), 
              React.createElement("li", {className: (location.pathname=="/admin/models" && "active")}, React.createElement("a", {href: "/admin/models"}, "Models")), 
              React.createElement("li", {className: (location.pathname=="/admin/servers" && "active")}, React.createElement("a", {href: "/admin/servers"}, "System"))
            )
          )
          );
    }
    return (
      React.createElement("div", null, 
        React.createElement("hr", null), 
            React.createElement("a", {href: "/"}, React.createElement("img", {alt: "Brand", src: "/img/logo-scienceops-with-text-white-bkg-clear.png", height: "40px", style: styles})), 
        React.createElement("hr", null), 
        React.createElement("ul", {className: "nav nav-sidebar"}, 
          React.createElement("li", {className: this.props.page=="overview" && "active"}, React.createElement("a", {href: "/"}, "Overview")), 
          React.createElement("li", {className: this.props.page=="account" && "active"}, React.createElement("a", {href: "/account"}, "Account")), 
          adminNav, 
          React.createElement("li", {className: this.props.page=="documentation" && "active"}, React.createElement("a", {target: "_blank", href: "http://help.yhathq.com"}, "Documentation")), 
          React.createElement("li", {className: this.props.page=="logout" && "active"}, React.createElement("a", {href: "/logout"}, "Logout"))
        ), 
        React.createElement("hr", null)
      )
    );
  }
})

// ErrorModal is to used to display errors stemming from bad requests.
// Use the globally available "showError()" function to trigger it.
var ErrorModal = React.createClass({displayName: "ErrorModal",
  getInitialState: function() {
    return { title: '', body: '' };
  },
  show: function(title, error) {
    this.setState({ title: title, body: error});
    $(React.findDOMNode(this.refs.modal)).modal('show');
  },
  render: function() {
    return (
      React.createElement("div", {ref: "modal", className: "modal fade"}, 
        React.createElement("div", {className: "modal-dialog"}, 
          React.createElement("div", {className: "modal-content"}, 
            React.createElement("div", {className: "modal-header"}, 
              React.createElement("button", {type: "button", ref: "button", className: "close", "data-dismiss": "modal", "aria-label": "Close"}, React.createElement("span", {"aria-hidden": "true"}, "×")), 
              React.createElement("h4", {className: "modal-title"}, this.state.title)
            ), 
            React.createElement("div", {className: "modal-body"}, 
              React.createElement("p", null, this.state.body)
            ), 
            React.createElement("div", {className: "modal-footer"}, 
              React.createElement("button", {type: "button", className: "btn btn-default", "data-dismiss": "modal"}, "Close")
            )
          )
        )
      )
    )
  }
})

var Highlight = React.createClass({displayName: "Highlight",
  render: function() {
    var code = this.props.code
                .replace("{USERNAME}", this.props.username).replace("{USERNAME}", this.props.username)
                .replace("{APIKEY}", this.props.apikey)
                .replace("{MODEL_NAME}", this.props.modelName)
                .replace("{DATA}", this.props.data)
                .replace("{DOMAIN}", this.props.domain)

    var highlightStyle;
    if(this.props.highlightStyle){
      highlightStyle = this.props.highlightStyle;
    } else {
      highlightStyle = {fontSize: 14};
    }
    return (
      React.createElement("pre", null, React.createElement("code", {style: highlightStyle, className: this.props.lang}, code))
    );
  }
});