var DeployBtn = React.createClass({
  redeploy: function() {
     var path = "/models/" + this.props.modelname + "/redeploy/" + this.props.version;
     $.ajax({
       method: 'POST',
       url: path,
       success: function(data) {
       }.bind(this),
       error: function(xhr, status, err) {
         console.error(this.props.url, status, err.toString());
       }.bind(this)
     });
     
  },
  render: function() {
    return (
      <button onClick={this.redeploy} className="btn btn-xs btn-default">
        Deploy Version
      </button>
    )
  }
});

var VersionRow = React.createClass({
  showSourceCode: function() {
    var modal = $("#sourceCodeModal");
    modal.modal('hide');
    $("#sourceCode").text(this.props.version.Code.trim());
    modal.modal('show');
  },
  render: function() {
    return (
      <tr>
        <td>{this.props.version.Version}</td>
        <td>{this.props.version.CreatedAt}</td>
        <td>
          <button className="btn btn-xs btn-default" onClick={this.showSourceCode}>
            Show Source
         </button>
        </td>
        <td>
          <DeployBtn modelname={this.props.modelname} version={this.props.version.Version}/>
        </td>
      </tr>
    )
  }
});

var VersionTable = React.createClass({

    getInitialState: function() { return {versions: [], page: 1, query: ""}; },

    incrementPageCount: function(e) { this.setState({page: this.state.page + 1}); },

    decrementPageCount: function(e) { this.setState({page: this.state.page - 1}); },
    setPageCount: function(pageCount) { this.setState({ page: pageCount }); },

    setQuery: function(event) {
      this.setState({ query: event.target.value, page: 1 });
    },

    getFromServer: function() {
      var urlPath = "/models/" + this.props.modelname + "/versions.json";
      $.ajax({
        url: urlPath,
        dataType: 'json',
        success: function(data) {
          data.sort(function(v1, v2) {
            return v2.Version - v1.Version;
          });
          if (data.length != this.state.versions.length) {
            this.setState({ versions: data });
          }
        }.bind(this),
        error: function(xhr, status, err) {
          console.error(this.props.url, status, err.toString());
        }.bind(this)
      });
    },

    componentDidMount: function() {
      this.getFromServer();
      setInterval(this.getFromServer, 1000);
    },

    render: function() {
      var queries = this.state.query.split(/\s+/)
      var matchesQuery = function(term) {
        for(var i = 0; i < queries.length; i++) {
          var query = queries[i];
          if (term.indexOf(query) > -1) {
            return true;
          }
        }
        return false;
      }

      var filtered = [];
      for (var i = 0; i < this.state.versions.length; i++) {
        var version = this.state.versions[i];
        if (matchesQuery(version.Version.toString()) || matchesQuery(version.CreatedAt)) {
          filtered.push(version);
        }
      }
      var n = filtered.length;
      var nPages = Math.ceil(n / 10);
      var currPage = this.state.page;

      var leftDisabled = "disabled"
      var rightDisabled = "disabled"

      if(currPage > 1) {
        leftDisabled = ""
      }
      if(currPage < pages) {
        rightDisabled = ""
      }

      var pages = []
      if(nPages === 1) {
        pages = [1]
      } else if (nPages <= 10) {
        for(i = 0; i < nPages; i++) {
          pages.push(i+1);
        }
      } else {
         // handle cases where there are more than 10 pages.
         // push first page
         pages.push(1);

         // start at currPage - 4 or 2, whichever is larger.
         var i = currPage - 4;
         if (i < 2) {
            i = 2;
         }
         // go until currPage
         for ( ; i < currPages; i++) {
            pages.push(i);
         }

         pages.push(currPage);

         // push 4 more pages
         var j = currPage + 1;
         for ( ; (j < nPages) && (j < currPage + 4); j++) {
            pages.push(j);
         }

         pages.push(nPages);
      }


      var first = (currPage - 1) * 10;
      var last = currPage * 10;
      var toShow = filtered.slice(first, last);

      var setPageCount = this.setPageCount;
      var setPage = function(page) {
          return function(){ setPageCount(page); }
      }
      var modelname = this.props.modelname;
      return (
        <div>
        <div>
            <input type="text" className="form-control" onChange={this.setQuery} placeholder="Search">
            </input>
        </div>
        <div className="col-md-12">
          <table className="table">
              <thead>
                  <tr>
                      <th className="col-md-2">Version Number</th>
                      <th className="col-md-3">Date Deployed</th>
                      <th className="col-md-2">Source Code</th>
                      <th className="col-md-1">Deploy</th>
                  </tr>
              </thead>
              <tbody>
                {toShow.map(function(version){
                  return <VersionRow version={version} modelname={modelname} /> 
                })}
              </tbody>
          </table>
          <nav className="col-md-offset-5">
            <ul className="pagination">

              <li className={leftDisabled}>
                <a href="#" aria-label="Previous" onClick={this.decrementPageCount}>
                <span aria-hidden="true">&laquo;</span>
              </a></li>                    
              {pages.map(function(page){
                var cName = (page === currPage) ? "disabled" : "";
                return <li className={cName}>
                    <a href="#" onClick={setPage(page)}>{page}</a>
                </li>
              })}
              <li className={rightDisabled}>
                <a href="#" aria-label="Next" onClick={this.incrementPageCount}>
                <span aria-hidden="true">&raquo;</span>
               </a></li>
            </ul>
          </nav>                
        </div>
        </div>
      );
    }
});

var modelname = window.location.pathname.split("/")[2];

React.render(<VersionTable modelname={modelname}/>, 
        document.getElementById("versions-table"));
React.render(<ModelHeader url={window.location.pathname.split("/").slice(0, 3).join("/") + "/json"} />, document.getElementById("model-header"));
React.render(<ModelNav page="versions" route={window.location.pathname.split("/").slice(0, 3).join("/")} />, document.getElementById("model-nav"));
