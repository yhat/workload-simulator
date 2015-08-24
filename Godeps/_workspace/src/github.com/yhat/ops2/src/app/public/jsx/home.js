var ModelRow = React.createClass({
    render: function() {
      var url = "/models/" + this.props.model.Name + "/scoring";
      return (
          <tr>
            <td><a href={url}>{this.props.model.Name}</a></td>
            <td><FormattedDate value={this.props.model.LastUpdated} /></td>
            <td>{this.props.model.NumVersions}</td>
            <td><ModelStatus value={this.props.model.Status} /></td>
          </tr>
      );
    }
});

var ModelTable = React.createClass({
  getInitialState: function(){
    return { user: {}};    
  },
  componentWillMount: function() {
    // grab user data to make predictions with
    $.ajax({
      url: "/user.json",
      dataType: 'json',
      cache: false,
      async: false,
      success: function(data) {
        this.setState({ user: data});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  highlightCode: function() {
    $('pre code').each(function(i, block) {
      hljs.highlightBlock(block);
    });
  },
  render: function() {
      setTimeout(this.highlightCode(), 10);
      var rows = [];
      var lastCategory = null;
      this.props.models.forEach(function(model) {
        var searchText = [model.Name, model.LastUpdated, model.NumVersions].join("-").toLowerCase();
        var queries = this.props.filterText.trim().toLowerCase().split(' ');
        for(var i=0; i < queries.length; i++) {
          var pg = Math.floor((rows.length + 1) / 10);
          if (searchText.indexOf(queries[i]) >= 0) {
            rows.push(<ModelRow idx={rows.length + 1} model={model} key={model.name} />);
            break;
          }
          if (queries=='') {
            rows.push(<ModelRow idx={rows.length + 1} model={model} key={model.name} />);
            break;
          }
        }

      }.bind(this));
      var tooltipHelper = "<p class='text-left small'><span class='label label-danger'>Down</span> Model has either crashed or could not be deployed.</p>";
      tooltipHelper += "<p class='text-left small'><span class='label label-warning'>Offline</span> Model was deliberately shutdown.</p>";
      tooltipHelper += "<p class='text-left small'><span class='label label-info'>Queued</span> Deployment has been initiated. Waiting for available resources for deployment.</p>";
      tooltipHelper += "<p class='text-left small'><span class='label label-primary'>Building</span> Model environment being setup.</p>";
      tooltipHelper += "<p class='text-left small'><span class='label label-success'>Online</span> Model deployed successfully and accepting requests.</p>";

      var domain = window.location.origin;

      if (rows.length > 0){
        return (
            <table className="table">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Last Updated</th>
                        <th>Versions</th>
                        <th>Status <span data-toggle="tooltip" data-html="true" data-placement="bottom" title={tooltipHelper} className="fa fa-question-circle"></span></th>
                    </tr>
                </thead>
                <tbody>{rows}</tbody>
            </table>
        )
      } else {
        return (
            <div>
            Welcome! You have not deployed a model yet. Try deploying one by following the examples below.
            <div className="row">
              <div className="col-sm-6">
                <h4>R</h4>
                <div id="r-hello-template" style={{height: 600}}>
                  <Highlight lang="R" domain={domain} code={DeploySamples.R} username={this.state.user.Name} apikey={this.state.user.Apikey} data={this.state.modelInput} highlightStyle={{fontSize: 14, height: 350}}/>
                </div>
              </div>
              <div class="col-sm-6">
                <h4>Python</h4>
                <div id="py-hello-template" style={{height: 600}}>
                  <Highlight lang="python" domain={domain} code={DeploySamples.Python} username={this.state.user.Name} apikey={this.state.user.Apikey} data={this.state.modelInput} highlightStyle={{fontSize: 14, height: 350}}/>
                </div>
              </div>
            </div>
            </div>
          )
      }
    }
});

var SearchBar = React.createClass({
    handleChange: function() {
        this.props.onUserInput(
            this.refs.filterTextInput.getDOMNode().value
        );
    },
    render: function() {
        return (
            <form>
              <div className="form-group">
                <div className="input-group">
                  <div className="input-group-addon"><i className="fa fa-search"></i></div>
                  <input
                      className="form-control"
                      type="text"
                      placeholder="Search..."
                      value={this.props.filterText}
                      ref="filterTextInput"
                      onChange={this.handleChange}
                  />
                </div>
              </div>
            </form>
        );
    }
});

var FilterableModelTable = React.createClass({
    getInitialState: function() {
        return {
            filterText: '',
            models: [],
            firstRender: true
        };
    },
    getModels: function() {
      $.ajax({
        url: this.props.url,
        dataType: 'json',
        cache: false,
        success: function(data) {
          this.setState({"models": data, firstRender: false});
        }.bind(this),
        error: function(xhr, status, err) {
          console.error(this.props.url, status, err.toString());
        }.bind(this)
      });
    },
    componentWillMount: function(){
        this.getModels();
        setInterval(this.getModels, this.props.pollInterval);
    },
    handleUserInput: function(filterText) {
        this.setState({
            filterText: filterText
        });
    },
    shouldComponentUpdate: function(nextState) {
        if (this.state.firstRender) {
            return true;
        }
        isLenZero = function(models) {
            if (models === undefined) {
                return true;
            }
            return models.length === 0;
        }
        if (isLenZero(this.state.models) && isLenZero(nextState.models)) {
            return false;
        }
        return true;
    },
    render: function() {
        return (
            <div>
                <SearchBar
                    filterText={this.state.filterText}
                    onUserInput={this.handleUserInput}
                />
                <h4>Models</h4>
                <ModelTable
                    models={this.state.models}
                    filterText={this.state.filterText}
                />
            </div>
        );
    }
});

var SharedModelTable = React.createClass({
  getInitialState: function() {
    return {models: []};
  },
  getSharedModels: function() {
    $.ajax({
      url: "shared",
      dataType: 'json',
      cache: false,
      success: function(data) {
        this.setState({"models": data});
      }.bind(this),
      error: function(xhr, status, err) {
        console.error(this.props.url, status, err.toString());
      }.bind(this)
    });
  },
  componentWillMount: function(){
    this.getSharedModels();
    setInterval(this.getSharedModels, this.props.pollInterval);
  },
  render: function() {
      if (this.state.models.length === 0) {
        return <div></div>
      }

      renderRow = function(model) {
        return (
          <tr>
            <td>{model.Owner}</td>
            <td>{model.Name}</td>
            <td>{model.LastUpdated}</td>
          </tr>
        )
      }

      return (
        <div>
          <h4>Shared with you <small>You can use your APIKey to query these</small></h4>
          <table className="table">
            <thead>
              <tr>
                <th>Model Owner</th>
                <th>Model Name</th>
                <th>Last Updated</th>
              </tr>
            </thead>
            <tbody>{this.state.models.map(renderRow)}</tbody>
          </table>
        </div>
      );
  }
});

React.render(<FilterableModelTable url="/models" pollInterval={1000} />, document.getElementById('search-bar'));
// enable all tooltips at once
$('[data-toggle="tooltip"]').tooltip();

React.render(<SharedModelTable pollInterval={1000} />, document.getElementById('shared-models'));
