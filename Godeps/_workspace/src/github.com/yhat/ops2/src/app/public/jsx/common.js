var ModelStatus = React.createClass({
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
        <span className={label_type}>build {this.props.value}</span>
    );
  }
});

var ModelHeader = React.createClass({
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
      img = <img src="/img/logo-python.png" height="38px" />
    } else if (this.state.version.Lang == "r") {
      img = <img src="/img/logo-r.png" height="38px" />      
    } else {
      img = <div style={{height: 38}}/>
    }
    return (
      <div>
        <div className="col-sm-2">
          <p className="text-muted"><a href="/"><i className="fa fa-angle-left"></i> Models</a></p>
        </div>
        <div className="col-sm-8">
          <h3 className="text-center">
             {img} {this.state.model.Name} <small>Version {this.state.model.ActiveVersion}</small>
          </h3>
          <p className="text-center">
            <ModelStatus value={this.state.model.Status} />
          </p>
        </div>
      </div>
    );
  }
});

var ModelNav = React.createClass({
  render: function() {
    return (
      <ul className="nav nav-pills nav-justified">
        <li className={this.props.page=="scoring" && "active"} role="presentation">
          <a href={this.props.route + "/scoring"}><i className="fa fa-crosshairs"></i> Scoring</a>
        </li>
        <li className={this.props.page=="versions" && "active"} role="presentation">
          <a href={this.props.route + "/versions"}><i className="fa fa-list-ol"></i> Versions</a>
        </li>
        <li className={this.props.page=="logs" && "active"} role="presentation">
          <a href={this.props.route + "/logs"}><i className="fa fa-terminal"></i> Logs</a>
        </li>
        <li className={this.props.page=="settings" && "active"} role="presentation">
          <a href={this.props.route + "/settings"}><i className="fa fa-cogs"></i> Settings</a>
        </li>
      </ul>
    );
  }
});

var AdminNav = React.createClass({
  render: function() {
    return (
      <ul className="nav nav-pills nav-justified">
        <li className={location.pathname=="/admin/users" && "active"} role="presentation">
          <a href="/admin/users"><i className="fa fa-users"></i> Users</a>
        </li>
        <li className={location.pathname=="/admin/models" && "active"} role="presentation">
          <a href="/admin/models"><i className="fa fa-list"></i> Models</a>
        </li>
        <li className={location.pathname=="/admin/servers" && "active"} role="presentation">
          <a href="/admin/servers"><i className="fa fa-server"></i> System</a>
        </li>
      </ul>
    );
  }
})

var FormattedDate = React.createClass({

  render: function() {
    var date = new Date(this.props.value);
    date = date.toLocaleDateString() + " " + date.toLocaleTimeString();
    return (
      <span>{date}</span>
    );
  }
});

var If = React.createClass({
    render: function() {
        if (this.props.cond) {
            return this.props.children;
        }
        else {
            return false;
        }
    }
});

var SideNav = React.createClass({
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
      adminNav = <li className={(this.state.user.Admin ? "" : " hide")}><a href="/admin/users">Admin</a></li>
    }
    if (this.props.page=="admin" && this.state.user.Admin) {
      adminNav = (
          <li className={(this.props.page=="admin" && "active")}><a href="/admin/users">Admin</a>
            <ul className="sub-nav">
              <li className={(location.pathname=="/admin/users" && "active")}><a href="/admin/users">Users</a></li>
              <li className={(location.pathname=="/admin/models" && "active")}><a href="/admin/models">Models</a></li>
              <li className={(location.pathname=="/admin/servers" && "active")}><a href="/admin/servers">System</a></li>
            </ul>
          </li>
          );
    }
    return (
      <div>
        <hr />
            <a href="/"><img alt="Brand" src="/img/logo-scienceops-with-text-white-bkg-clear.png" height="40px" style={styles}/></a>
        <hr />
        <ul className="nav nav-sidebar">
          <li className={this.props.page=="overview" && "active"}><a href="/">Overview</a></li>
          <li className={this.props.page=="account" && "active"}><a href="/account">Account</a></li>
          {adminNav}
          <li className={this.props.page=="documentation" && "active"}><a target="_blank" href="http://help.yhathq.com">Documentation</a></li>
          <li className={this.props.page=="logout" && "active"}><a href="/logout">Logout</a></li>
        </ul>
        <hr />
      </div>
    );
  }
})

// ErrorModal is to used to display errors stemming from bad requests.
// Use the globally available "showError()" function to trigger it.
var ErrorModal = React.createClass({
  getInitialState: function() {
    return { title: '', body: '' };
  },
  show: function(title, error) {
    this.setState({ title: title, body: error});
    $(React.findDOMNode(this.refs.modal)).modal('show');
  },
  render: function() {
    return (
      <div ref="modal" className="modal fade">
        <div className="modal-dialog">
          <div className="modal-content">
            <div className="modal-header">
              <button type="button" ref="button" className="close" data-dismiss="modal" aria-label="Close"><span aria-hidden="true">&times;</span></button>
              <h4 className="modal-title">{this.state.title}</h4>
            </div>
            <div className="modal-body">
              <p>{this.state.body}</p>
            </div>
            <div className="modal-footer">
              <button type="button" className="btn btn-default" data-dismiss="modal">Close</button>
            </div>
          </div>
        </div>
      </div>
    )
  }
})

var Highlight = React.createClass({
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
      <pre><code style={highlightStyle} className={this.props.lang}>{code}</code></pre>
    );
  }
});