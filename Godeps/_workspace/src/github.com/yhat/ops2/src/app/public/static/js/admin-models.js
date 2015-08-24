var ModelRow = React.createClass({displayName: "ModelRow",
    render: function() {
      console.log(this.props.model);
      var url = "/users/" + this.props.model.Owner + "/models/" + this.props.model.Name + "/scoring";
      return (
          React.createElement("tr", null, 
            React.createElement("td", null, this.props.model.Owner), 
            React.createElement("td", null, React.createElement("a", {href: url}, this.props.model.Name)), 
            React.createElement("td", null, React.createElement(FormattedDate, {value: this.props.model.LastUpdated})), 
            React.createElement("td", null, this.props.model.NumVersions), 
            React.createElement("td", null, React.createElement(ModelStatus, {value: this.props.model.Status}))
          )
      );
    }
});

var ModelTable = React.createClass({displayName: "ModelTable",
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
            rows.push(React.createElement(ModelRow, {idx: rows.length + 1, model: model, key: model.name}));
            break;
          }
          if (queries=='') {
            rows.push(React.createElement(ModelRow, {idx: rows.length + 1, model: model, key: model.name}));
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


      return (
          React.createElement("table", {className: "table"}, 
              React.createElement("thead", null, 
                  React.createElement("tr", null, 
                      React.createElement("th", null, "Owner"), 
                      React.createElement("th", null, "Name"), 
                      React.createElement("th", null, "Last Updated"), 
                      React.createElement("th", null, "Versions"), 
                      React.createElement("th", null, "Status ", React.createElement("span", {"data-toggle": "tooltip", "data-html": "true", "data-placement": "bottom", title: tooltipHelper, className: "fa fa-question-circle"}))
                  )
              ), 
              React.createElement("tbody", null, rows)
          )
      )
    }
});

var SearchBar = React.createClass({displayName: "SearchBar",
    handleChange: function() {
        this.props.onUserInput(
            this.refs.filterTextInput.getDOMNode().value
        );
    },
    render: function() {
        return (
            React.createElement("form", null, 
              React.createElement("div", {className: "form-group"}, 
                React.createElement("div", {className: "input-group"}, 
                  React.createElement("div", {className: "input-group-addon"}, React.createElement("i", {className: "fa fa-search"})), 
                  React.createElement("input", {
                      className: "form-control", 
                      type: "text", 
                      placeholder: "Search...", 
                      value: this.props.filterText, 
                      ref: "filterTextInput", 
                      onChange: this.handleChange}
                  )
                )
              )
            )
        );
    }
});

var FilterableModelTable = React.createClass({displayName: "FilterableModelTable",
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
            React.createElement("div", null, 
                React.createElement(SearchBar, {
                    filterText: this.state.filterText, 
                    onUserInput: this.handleUserInput}
                ), 
                React.createElement(ModelTable, {
                    models: this.state.models, 
                    filterText: this.state.filterText}
                )
            )
        );
    }
});

var SharedModelTable = React.createClass({displayName: "SharedModelTable",
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
        return React.createElement("div", null)
      }

      renderRow = function(model) {
        return (
          React.createElement("tr", null, 
            React.createElement("td", null, model.Owner), 
            React.createElement("td", null, model.Name), 
            React.createElement("td", null, model.LastUpdated)
          )
        )
      }

      return (
        React.createElement("div", null, 
          React.createElement("h4", null, "Shared with you ", React.createElement("small", null, "You can use your APIKey to query these")), 
          React.createElement("table", {className: "table"}, 
            React.createElement("thead", null, 
              React.createElement("tr", null, 
                React.createElement("th", null, "Model Owner"), 
                React.createElement("th", null, "Model Name"), 
                React.createElement("th", null, "Last Updated")
              )
            ), 
            React.createElement("tbody", null, this.state.models.map(renderRow))
          )
        )
      );
  }
});

React.render(React.createElement(FilterableModelTable, {url: "/admin/models.json", pollInterval: 1000}), document.getElementById('search-bar'));
// enable all tooltips at once
$('[data-toggle="tooltip"]').tooltip();
React.render(React.createElement(AdminNav, {page: "admin", route: window.location.pathname}), document.getElementById("admin-nav"));