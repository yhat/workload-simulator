var Apikey = React.createClass({
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
      <p>{this.props.text}: <code><span>{ this.state.apikey }</span></code></p>
    );
  }
});

var UserInfo = React.createClass({
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
    return <span><code>{this.state.username}</code></span>
  }  
});

var RegenerateApiKey = React.createClass({
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
        <p>{this.props.text}: <code><Apikey apikey_fieldname={this.props.apikey_fieldname} /></code></p>
      );
    } else if (this.state.displayed=="validating") {
      return (
        <span>
          <button onClick={this.handleClick} className="btn btn-default text-uppercase">regenerate {this.props.text}</button>
          <PasswordModal postValidation={this.postValidation} btnMessage="Regenerate" message="To regenerate your API key, enter your password:"/>
        </span>
      );
    } else if (this.state.displayed==false) {
      return (
        <button onClick={this.handleClick} className="btn btn-default text-uppercase">Regenerate {this.props.text}</button>
      );
    }
  }
});

var PasswordModal = React.createClass({
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
      alert = <div className="alert alert-warning" role="alert">{this.state.status}</div>;
    }
    return (
      <div className="modal fade password-modal" tabIndex="-1" role="dialog" aria-labelledby="validatePasswordModal">
        <div className="modal-dialog modal-md">
          <div className="modal-content">
            <div className="modal-header">
              <h4 className="text-primary text-center">{this.props.btnMessage} Key</h4>
            </div>
            <div className="modal-body">
              {alert}
              <form onSubmit={this.handleSubmit}>
                <div className="form-group">
                  <p className="small">{this.props.message}</p>
                  <input type="password" className="form-control" id="password" ref="password" placeholder="password" />
                </div>
                <div className="form-group text-right">
                  <button type="button" className="btn btn-default" data-dismiss="modal">Cancel</button>
                  &nbsp;
                  <button type="submit" className="btn btn-primary">{this.props.btnMessage}</button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>
    );
  }
});

//show username
React.render(<UserInfo/>, document.getElementById('username'));
// show current APIKEY

React.render(<Apikey apikey_fieldname="apikey" text="API KEY"/>, document.getElementById('api-key'));
React.render(<Apikey apikey_fieldname="read_only_apikey" text="READ-ONLY API KEY"/>, document.getElementById('read-only-key'));
// regenerates
React.render(<RegenerateApiKey apikey_fieldname="apikey" text="api key"/>, document.getElementById('api-key-regenerate'));
React.render(<RegenerateApiKey apikey_fieldname="read_only_apikey" text="read-only api key"/>, document.getElementById('read-only-key-regenerate'));

React.render(<SideNav page="account" />, document.getElementById('side-nav'));